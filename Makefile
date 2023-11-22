c_api:
	rm -rf build/api.so || true
	mkdir -p build || true
	go build -v -buildmode=c-shared -o build/api.so .

.PHONY: c_api