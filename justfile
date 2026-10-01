default:
    go run .

build:
    go build -o wrestling .

build-web:
    GOOS=js GOARCH=wasm go build -o dist/web/wrestling.wasm .
    cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/web/

serve: build-web
    cd dist/web && python3 -m http.server 8080
