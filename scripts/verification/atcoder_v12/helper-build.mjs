// helperのビルド条件。prepare.mjsと固定入力testが同じ値を使うため、ここだけで持つ。
//
// -buildvcs=false: Goはcommit idとdirty状態をバイナリへ埋める。付けないと、
// helperのsourceが同じでもcommitが進むだけでhashが変わる。campaign manifestは
// helperのhashを「挙動が変わったか」の判定に使うため、埋め込みがあると判定が
// 代理として機能しなくなり、無関係な変更でcampaignが無効になる。
// 判断の経緯はADR-0006を参照する。
export const HELPER_BUILD_FLAGS = ["-trimpath", "-buildvcs=false"];

export const HELPER_BUILD_ENV = { CGO_ENABLED: "0", GOOS: "darwin", GOARCH: "arm64" };
