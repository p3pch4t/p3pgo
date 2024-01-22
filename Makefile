NDK_HOME := $(or $(NDK_HOME),"/home/user/Android/Sdk/ndk/26.1.10909125/toolchains/llvm/prebuilt/linux-x86_64/bin")
ANDROID_ARM := "armv7a-linux-androideabi21-clang"
ANDROID_ARM64 := "aarch64-linux-android21-clang"
ANDROID_386 := "i686-linux-android21-clang"
ANDROID_AMD64 := "x86_64-linux-android21-clang"

all: clean c_api_android c_api_linux

c_api:
	-rm -rf build/api_host.so
	CGO_ENABLED=1 go build -v -buildmode=c-shared -o build/api_host.so .

c_api_android:
	CGO_ENABLED=1 CC=${NDK_HOME}/${ANDROID_ARM} CXX=${NDK_HOME}/${ANDROID_ARM}++ GOOS=android GOARCH=arm go build -v -buildmode=c-shared -o build/api_android_armeabi-v7a.so .
	CGO_ENABLED=1 CC=${NDK_HOME}/${ANDROID_ARM64} CXX=${NDK_HOME}/${ANDROID_ARM64}++ GOOS=android GOARCH=arm64 go build -v -buildmode=c-shared -o build/api_android_arm64-v8a.so .
	CGO_ENABLED=1 CC=${NDK_HOME}/${ANDROID_386} CXX=${NDK_HOME}/${ANDROID_386}++ GOOS=android GOARCH=386 go build -v -buildmode=c-shared -o build/api_android_x86.so .
	CGO_ENABLED=1 CC=${NDK_HOME}/${ANDROID_AMD64} CXX=${NDK_HOME}/${ANDROID_AMD64}++ GOOS=android GOARCH=amd64 go build -v -buildmode=c-shared -o build/api_android_x86_64.so .

c_api_linux:
	CGO_ENABLED=1 CC=i686-linux-gnu-gcc CXX=i386-linux-gnu-g++ GOOS=linux GOARCH=386 go build -v -buildmode=c-shared -o build/api_linux_i386.so .
	CGO_ENABLED=1 CC=x86_64-linux-gnu-gcc CXX=x86_64-linux-gnu-g++ GOOS=linux GOARCH=amd64 go build -v -buildmode=c-shared -o build/api_linux_amd64.so .
	CGO_ENABLED=1 CC=arm-linux-gnueabihf-gcc CXX=arm-linux-gnueabihf-g++ GOOS=linux GOARCH=arm go build -v -buildmode=c-shared -o build/api_linux_armhf.so .
	CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ GOOS=linux GOARCH=arm64 go build -v -buildmode=c-shared -o build/api_linux_aarch64.so .

clean:
	-rm -rf build

.PHONY: c_api clean c_api_android c_api_linux