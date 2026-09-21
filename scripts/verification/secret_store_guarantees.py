#!/usr/bin/env python3
"""OSの秘密情報保管庫が実際に保証する範囲を観測する。

`TD-12`と`TD-46`が必要とする2点を、製品が使うものと同じAPIで確認する。

- 同じ利用者として動作する他のプロセスから読めるか
- AlgoLoomの更新時に再認可を求められるか

使い捨ての固定値だけを保存し、観測後に削除する。AtCoderへ接続せず、
実アカウント、Cookie、認証情報を一切扱わない。出力は端末名、利用者名、
ホームディレクトリ配下の絶対パスを伏せ、そのまま記録へ貼れるようにする。

    python3 scripts/verification/secret_store_guarantees.py
    python3 scripts/verification/secret_store_guarantees.py --self-test
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import re
import secrets
import subprocess
import sys
import tempfile
from pathlib import Path

PROBE_PREFIX = "algoloom-secret-store-probe-"
PROBE_ACCOUNT = "probe"
PROBE_VALUE = "PROBE-VALUE-NOT-A-SECRET"
SUBPROCESS_TIMEOUT_SECONDS = 15

# 観測の種類。macOS、Windows、Linuxで同じIDを使い、結果を並べて比較できるようにする。
OBSERVATIONS = (
    ("same-process", "作成したプロセス自身から読めるか"),
    ("same-interpreter", "同じインタプリタの別スクリプトから読めるか（AlgoLoomの更新に相当）"),
    ("other-executable", "別の実行ファイルから読めるか（同じ利用者の他プロセスに相当）"),
    ("delete", "読み出しの認可なしで削除できるか"),
    ("residue", "後始末のあと残っていないか"),
)

READABLE = "読めた"
NOT_READABLE = "読めなかった"
# 「読めなかった」と「判定できなかった」を分ける。観測物の不具合で値を取り出せなかった場合を
# 保管庫の保護として記録すると、その上に過大な表示文言を置くことになる（認証設計 §4.1.2.1）。
UNDETERMINED = "判定できなかった"
DELETED = "削除できた"
PROMPTED = "確認画面が出た（対話が要る）"
SKIPPED = "実施できず"
REMOVED = "残っていない"


def redact(text: str) -> str:
    """端末名、利用者名、ホーム配下の絶対パスを伏せる。記録へ貼れる出力にするため。"""
    home = str(Path.home())
    out = text.replace(home, "<HOME>")
    out = re.sub(r"/(?:Users|home)/[^/\s\"']+", "<HOME>", out)
    out = re.sub(r"[A-Za-z]:\\\\Users\\\\[^\\\\\s\"']+", "<HOME>", out)
    out = re.sub(r"[A-Za-z]:\\Users\\[^\\\s\"']+", "<HOME>", out)
    user = os.environ.get("USER") or os.environ.get("USERNAME")
    if user and len(user) >= 3:
        out = out.replace(user, "<USER>")
    node = platform.node()
    if node and len(node) >= 3:
        out = out.replace(node, "<HOST>")
    return out


def new_probe_name() -> str:
    return PROBE_PREFIX + secrets.token_hex(4)


def is_probe_name(name: str) -> bool:
    """保管庫へ触れる前の安全弁。probe専用の名前以外は絶対に扱わない。"""
    return bool(re.fullmatch(re.escape(PROBE_PREFIX) + r"[0-9a-f]{8}", name))


def run_child(argv: list[str], undetermined_exit_codes: tuple[int, ...] = ()) -> tuple[str, str]:
    """子プロセスを有限時間で実行する。timeoutは対話の確認画面が出たものとして扱う。

    `undetermined_exit_codes`は、子プロセスが「読めなかった」ではなく「判定できなかった」を
    伝える終了コードである。呼び出し側が自分で決めた規約だけを渡す。
    """
    try:
        done = subprocess.run(argv, capture_output=True, text=True,
                              timeout=SUBPROCESS_TIMEOUT_SECONDS)
    except subprocess.TimeoutExpired:
        return PROMPTED, f"{SUBPROCESS_TIMEOUT_SECONDS}秒以内に終わらなかった"
    except OSError as error:
        return SKIPPED, f"起動できない: {error.__class__.__name__}"
    output = (done.stdout + done.stderr).strip()
    if done.returncode in undetermined_exit_codes:
        return UNDETERMINED, output
    return (READABLE if done.returncode == 0 and PROBE_VALUE in output else NOT_READABLE), output


class Backend:
    """OSごとの保管庫。read/write/deleteと、別の実行ファイルからの読み出しを提供する。"""

    name = "未対応"

    def write(self, target: str, value: str) -> str: raise NotImplementedError
    def read(self, target: str) -> tuple[bool, str]: raise NotImplementedError
    def delete(self, target: str) -> str: raise NotImplementedError
    def read_from_other_executable(self, target: str) -> tuple[str, str]: raise NotImplementedError


class MacKeychainBackend(Backend):
    """macOS Keychain。`keyring`のmacOS backendと同じSecItem APIを使う。"""

    name = "macOS Keychain（login keychain）"

    def __init__(self) -> None:
        import ctypes
        import ctypes.util

        self.ctypes = ctypes
        self.sec = ctypes.CDLL(ctypes.util.find_library("Security"))
        self.cf = ctypes.CDLL(ctypes.util.find_library("CoreFoundation"))
        c_void_p, c_int32, c_long = ctypes.c_void_p, ctypes.c_int32, ctypes.c_long
        self.cf.CFStringCreateWithCString.restype = c_void_p
        self.cf.CFStringCreateWithCString.argtypes = [c_void_p, ctypes.c_char_p, ctypes.c_uint32]
        self.cf.CFDataCreate.restype = c_void_p
        self.cf.CFDataCreate.argtypes = [c_void_p, ctypes.c_char_p, c_long]
        self.cf.CFDataGetBytePtr.restype = ctypes.POINTER(ctypes.c_char)
        self.cf.CFDataGetBytePtr.argtypes = [c_void_p]
        self.cf.CFDataGetLength.restype = c_long
        self.cf.CFDataGetLength.argtypes = [c_void_p]
        self.cf.CFDictionaryCreate.restype = c_void_p
        self.cf.CFDictionaryCreate.argtypes = [
            c_void_p, ctypes.POINTER(c_void_p), ctypes.POINTER(c_void_p), c_long, c_void_p, c_void_p]
        for fn in ("SecItemAdd", "SecItemCopyMatching"):
            handle = getattr(self.sec, fn)
            handle.restype, handle.argtypes = c_int32, [c_void_p, ctypes.POINTER(c_void_p)]
        self.sec.SecItemDelete.restype, self.sec.SecItemDelete.argtypes = c_int32, [c_void_p]
        self.key_cb = ctypes.cast(
            ctypes.addressof(c_void_p.in_dll(self.cf, "kCFTypeDictionaryKeyCallBacks")), c_void_p)
        self.value_cb = ctypes.cast(
            ctypes.addressof(c_void_p.in_dll(self.cf, "kCFTypeDictionaryValueCallBacks")), c_void_p)

    def _const(self, name: str, lib=None):
        return self.ctypes.c_void_p.in_dll(lib or self.sec, name).value

    def _string(self, text: str):
        return self.cf.CFStringCreateWithCString(None, text.encode(), 0x08000100)

    def _query(self, target: str, *, value: str | None = None, want_data: bool = False):
        pairs = [(self._const("kSecClass"), self._const("kSecClassGenericPassword")),
                 (self._const("kSecAttrService"), self._string(target)),
                 (self._const("kSecAttrAccount"), self._string(PROBE_ACCOUNT))]
        if value is not None:
            pairs.append((self._const("kSecValueData"),
                          self.cf.CFDataCreate(None, value.encode(), len(value.encode()))))
        if want_data:
            pairs.append((self._const("kSecReturnData"), self._const("kCFBooleanTrue", self.cf)))
            pairs.append((self._const("kSecMatchLimit"), self._const("kSecMatchLimitOne")))
        count = len(pairs)
        keys = (self.ctypes.c_void_p * count)(*[p[0] for p in pairs])
        values = (self.ctypes.c_void_p * count)(*[p[1] for p in pairs])
        return self.cf.CFDictionaryCreate(None, keys, values, count, self.key_cb, self.value_cb)

    def write(self, target: str, value: str) -> str:
        return f"SecItemAdd rc={self.sec.SecItemAdd(self._query(target, value=value), None)}"

    def read(self, target: str) -> tuple[bool, str]:
        out = self.ctypes.c_void_p()
        rc = self.sec.SecItemCopyMatching(self._query(target, want_data=True), self.ctypes.byref(out))
        if rc != 0:
            return False, f"SecItemCopyMatching rc={rc}"
        data = self.ctypes.string_at(self.cf.CFDataGetBytePtr(out), self.cf.CFDataGetLength(out))
        return data.decode() == PROBE_VALUE, "SecItemCopyMatching rc=0"

    def delete(self, target: str) -> str:
        return f"SecItemDelete rc={self.sec.SecItemDelete(self._query(target))}"

    def read_from_other_executable(self, target: str) -> tuple[str, str]:
        # `/usr/bin/security`はApple署名の別の実行ファイル。ACLが許可しない場合は確認画面が出る。
        status, detail = run_child(
            ["/usr/bin/security", "find-generic-password", "-w", "-s", target, "-a", PROBE_ACCOUNT])
        if status == PROMPTED:
            detail += "。画面に確認ダイアログが残っていれば閉じてください"
        return status, detail


class WindowsCredentialLockerBackend(Backend):
    """WindowsのOS保護領域。`keyring`のWindows backendと同じCredential Manager APIを使う。"""

    name = "WindowsのOS保護領域（Credential Manager、CRED_TYPE_GENERIC）"

    CRED_TYPE_GENERIC = 1
    CRED_PERSIST_LOCAL_MACHINE = 2

    def __init__(self) -> None:
        import ctypes
        from ctypes import wintypes

        self.ctypes = ctypes

        class FILETIME(ctypes.Structure):
            _fields_ = [("dwLowDateTime", wintypes.DWORD), ("dwHighDateTime", wintypes.DWORD)]

        class CREDENTIAL(ctypes.Structure):
            _fields_ = [
                ("Flags", wintypes.DWORD), ("Type", wintypes.DWORD),
                ("TargetName", wintypes.LPWSTR), ("Comment", wintypes.LPWSTR),
                ("LastWritten", FILETIME), ("CredentialBlobSize", wintypes.DWORD),
                ("CredentialBlob", ctypes.POINTER(ctypes.c_byte)), ("Persist", wintypes.DWORD),
                ("AttributeCount", wintypes.DWORD), ("Attributes", ctypes.c_void_p),
                ("TargetAlias", wintypes.LPWSTR), ("UserName", wintypes.LPWSTR)]

        self.CREDENTIAL = CREDENTIAL
        self.advapi32 = ctypes.WinDLL("advapi32", use_last_error=True)
        self.advapi32.CredWriteW.argtypes = [ctypes.POINTER(CREDENTIAL), wintypes.DWORD]
        self.advapi32.CredWriteW.restype = wintypes.BOOL
        self.advapi32.CredReadW.argtypes = [
            wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD, ctypes.POINTER(ctypes.POINTER(CREDENTIAL))]
        self.advapi32.CredReadW.restype = wintypes.BOOL
        self.advapi32.CredDeleteW.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD]
        self.advapi32.CredDeleteW.restype = wintypes.BOOL
        self.advapi32.CredFree.argtypes = [ctypes.c_void_p]

    def write(self, target: str, value: str) -> str:
        blob = value.encode("utf-16-le")
        buffer = self.ctypes.create_string_buffer(blob, len(blob))
        cred = self.CREDENTIAL()
        cred.Type = self.CRED_TYPE_GENERIC
        cred.TargetName = target
        cred.UserName = PROBE_ACCOUNT
        cred.CredentialBlobSize = len(blob)
        cred.CredentialBlob = self.ctypes.cast(buffer, self.ctypes.POINTER(self.ctypes.c_byte))
        cred.Persist = self.CRED_PERSIST_LOCAL_MACHINE
        ok = self.advapi32.CredWriteW(self.ctypes.byref(cred), 0)
        return f"CredWriteW ok={bool(ok)} lasterror={self.ctypes.get_last_error() if not ok else 0}"

    def read(self, target: str) -> tuple[bool, str]:
        pointer = self.ctypes.POINTER(self.CREDENTIAL)()
        ok = self.advapi32.CredReadW(target, self.CRED_TYPE_GENERIC, 0, self.ctypes.byref(pointer))
        if not ok:
            return False, f"CredReadW ok=False lasterror={self.ctypes.get_last_error()}"
        try:
            found = pointer.contents
            data = self.ctypes.string_at(found.CredentialBlob, found.CredentialBlobSize)
            return data.decode("utf-16-le") == PROBE_VALUE, "CredReadW ok=True"
        finally:
            self.advapi32.CredFree(pointer)

    def delete(self, target: str) -> str:
        ok = self.advapi32.CredDeleteW(target, self.CRED_TYPE_GENERIC, 0)
        return f"CredDeleteW ok={bool(ok)} lasterror={self.ctypes.get_last_error() if not ok else 0}"

    def read_from_other_executable(self, target: str) -> tuple[str, str]:
        # PowerShellはpython.exeとは別の実行ファイル。同じ利用者として動作する他のプロセスにあたる。
        script = (
            "$ErrorActionPreference='Stop';"
            "Add-Type -Namespace P -Name C -MemberDefinition '"
            "[DllImport(\"advapi32.dll\", CharSet=CharSet.Unicode, SetLastError=true)]"
            " public static extern bool CredReadW(string t, uint y, uint f, out IntPtr c);"
            "[DllImport(\"advapi32.dll\")] public static extern void CredFree(IntPtr c);';"
            "$p=[IntPtr]::Zero;"
            # CREDENTIAL構造体の位置はポインタ幅で変わる。64 bitでCredentialBlobSizeが32、
            # CredentialBlobが40、32 bitでは24と28。固定値にすると別の幅で空文字を読み、
            # 「読めなかった」と取り違える。
            "if([IntPtr]::Size -eq 8){$so=32;$bo=40}else{$so=24;$bo=28};"
            f"if([P.C]::CredReadW('{target}',1,0,[ref]$p))"
            "{$size=[Runtime.InteropServices.Marshal]::ReadInt32($p,$so);"
            "$ptr=[Runtime.InteropServices.Marshal]::ReadIntPtr($p,$bo);"
            "$s=[Runtime.InteropServices.Marshal]::PtrToStringUni($ptr,$size/2);"
            "[P.C]::CredFree($p);"
            # 読み出し自体は成功したのに値が空なら、保管庫の結果ではなく観測物の不具合。
            # 黙って「読めなかった」にせず、区別できる文字列を出す。
            "if([string]::IsNullOrEmpty($s))"
            "{Write-Output \"CredReadW succeeded but blob was empty (size=$size)\";exit 2};"
            "Write-Output $s;exit 0}"
            "else{Write-Output 'CredReadW failed';exit 1}")
        for shell in ("powershell.exe", "pwsh.exe"):
            # 終了コード2は、読み出しに成功したのに値が空だった場合である。保管庫の保護ではなく
            # 観測物の不具合であるため、「読めなかった」へ畳まず「判定できなかった」を返す。
            status, detail = run_child(
                [shell, "-NoProfile", "-NonInteractive", "-Command", script],
                undetermined_exit_codes=(2,))
            if status != SKIPPED:
                return status, f"{shell}: {detail}"
        return SKIPPED, "PowerShellを起動できなかった"


def backend_for_current_os() -> Backend | None:
    if sys.platform == "darwin":
        return MacKeychainBackend()
    if sys.platform == "win32":
        return WindowsCredentialLockerBackend()
    return None


def read_from_same_interpreter(target: str) -> tuple[str, str]:
    """同じインタプリタで別のスクリプトファイルを動かす。AlgoLoomのコードだけが変わった状況にあたる。"""
    source = (
        "import sys\n"
        f"sys.path.insert(0, {str(Path(__file__).resolve().parent)!r})\n"
        "from secret_store_guarantees import backend_for_current_os, PROBE_VALUE\n"
        "backend = backend_for_current_os()\n"
        f"found, detail = backend.read({target!r})\n"
        "print(detail)\n"
        "print(PROBE_VALUE if found else 'not-found')\n"
        "sys.exit(0 if found else 1)\n")
    with tempfile.TemporaryDirectory() as workdir:
        child = Path(workdir) / "algoloom_probe_reader.py"
        child.write_text(source, encoding="utf-8")
        return run_child([sys.executable, str(child)])


def observe(skip_interactive: bool = False) -> dict:
    backend = backend_for_current_os()
    if backend is None:
        raise SystemExit(
            f"このOS（{sys.platform}）の保管庫はまだ実装していません。"
            "Linux Secret ServiceはTD-46で追加します。")

    target = new_probe_name()
    if not is_probe_name(target):
        raise SystemExit("probe名の生成に失敗しました")

    results: dict[str, dict[str, str]] = {}
    created = False
    try:
        write_detail = backend.write(target, PROBE_VALUE)
        found, read_detail = backend.read(target)
        created = found
        if not found:
            raise SystemExit(f"probeを保存できませんでした。{write_detail} / {read_detail}")
        results["same-process"] = {"結果": READABLE, "詳細": read_detail}
        status, detail = read_from_same_interpreter(target)
        results["same-interpreter"] = {"結果": status, "詳細": detail}
        if skip_interactive:
            results["other-executable"] = {"結果": SKIPPED, "詳細": "--skip-interactiveで省略した"}
        else:
            status, detail = backend.read_from_other_executable(target)
            results["other-executable"] = {"結果": status, "詳細": detail}
    finally:
        if created:
            delete_detail = backend.delete(target)
            still_there, _ = backend.read(target)
            results["delete"] = {
                "結果": DELETED if not still_there else "削除できなかった", "詳細": delete_detail}
            results["residue"] = {
                "結果": REMOVED if not still_there else "残っている", "詳細": "probe項目の再取得を確認"}

    return {
        "os": f"{platform.system()} {platform.release()}",
        "cpu": platform.machine(),
        "保管庫": backend.name,
        "実行ファイル": Path(sys.executable).name,
        "observations": results,
    }


def one_line(text: str, limit: int = 120) -> str:
    """表のセルへ入れるため、改行と連続空白を畳んで長さを切る。"""
    folded = " ".join(text.split())
    return folded if len(folded) <= limit else folded[: limit - 1] + "…"


def render(report: dict) -> str:
    lines = [
        "# 秘密情報保管庫の保証範囲の観測",
        "",
        f"- OS: {report['os']}",
        f"- CPU architecture: {report['cpu']}",
        f"- 保管庫: {report['保管庫']}",
        f"- 観測に使ったインタプリタ: {report['実行ファイル']}",
        "",
        "| 観測 | 内容 | 結果 | 詳細 |",
        "|---|---|---|---|",
    ]
    for key, description in OBSERVATIONS:
        found = report["observations"].get(key, {"結果": SKIPPED, "詳細": "―"})
        lines.append(f"| `{key}` | {description} | {found['結果']} | {one_line(found['詳細'])} |")
    return redact("\n".join(lines))


def self_test() -> int:
    """保管庫へ触れずに、安全弁と伏せ字と出力形式だけを確認する。"""
    checks = {
        "probe名の形式を検査する": is_probe_name(new_probe_name())
        and not is_probe_name("algoloom")
        and not is_probe_name(PROBE_PREFIX + "zzzzzzzz")
        and not is_probe_name("REVEL_SESSION"),
        # 検査用のpathをsource上へそのまま書かない。書くと`check-forbidden.mjs`が正しく落とす。
        "ホーム配下の絶対パスを伏せる": all(
            "<HOME>" in redact(sep.join(["", root, "example", "x"]))
            for root, sep in (("Users", "/"), ("home", "/")))
        and "<HOME>" in redact("C:" + "\\".join(["", "Users", "example", "x"])),
        "端末名と利用者名を伏せる": platform.node() not in redact(platform.node() or "x")
        if (platform.node() or "") and len(platform.node()) >= 3 else True,
        "観測の一覧が5件ある": len(OBSERVATIONS) == 5,
        "表のセルへ改行を入れない": "\n" not in one_line("a\nb") and one_line("a\nb") == "a b"
        and len(one_line("x" * 500)) == 120,
        "保存する値が秘密情報でない": "SECRET" not in PROBE_VALUE.replace("NOT-A-SECRET", ""),
        # 「読めなかった」と「判定できなかった」を取り違えると、実際には読めるものを
        # 読めないと記録し、その上に過大な表示文言を置いてしまう。
        "判定できなかったを読めなかったへ畳まない": run_child(
            [sys.executable, "-c", "import sys; sys.exit(2)"],
            undetermined_exit_codes=(2,))[0] == UNDETERMINED
        and run_child([sys.executable, "-c", "import sys; sys.exit(2)"])[0] == NOT_READABLE
        and run_child([sys.executable, "-c", "import sys; sys.exit(1)"],
                      undetermined_exit_codes=(2,))[0] == NOT_READABLE,
        "出力に全観測の行が出る": all(
            f"`{key}`" in render({"os": "x", "cpu": "y", "保管庫": "z",
                                  "実行ファイル": "w", "observations": {}})
            for key, _ in OBSERVATIONS),
    }
    for name, passed in checks.items():
        print(f"{'OK' if passed else 'NG'} {name}")
    return 0 if all(checks.values()) else 1


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--self-test", action="store_true",
                        help="保管庫へ触れずに安全弁と出力形式だけを確認する")
    parser.add_argument("--skip-interactive", action="store_true",
                        help="確認画面が出うる`other-executable`の観測を省く。再実行時の逃げ道")
    parser.add_argument("--json", action="store_true", help="観測結果をJSONで出す")
    args = parser.parse_args()
    if args.self_test:
        return self_test()
    report = observe(skip_interactive=args.skip_interactive)
    print(json.dumps(report, ensure_ascii=False, indent=2) if args.json else render(report))
    return 0


if __name__ == "__main__":
    sys.exit(main())
