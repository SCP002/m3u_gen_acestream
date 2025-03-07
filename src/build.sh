#!/bin/bash

project_name="m3u_gen_acestream"
main_file="${project_name}.go"
build_path="../build"

os_list=(
    "darwin"
    "freebsd"
    "linux"
    "netbsd"
    "openbsd"
    "windows"
)

arch_list=(
    "386"
    "amd64"
    "arm"
    "arm64"
    "loong64"
    "mips"
    "mips64"
    "mips64le"
    "mipsle"
    "s390"
    "s390x"
)

for os in "${os_list[@]}"; do
    if [[ $os == "windows" ]]; then
        extension=".exe"
    else
        extension=""
    fi

    for arch in "${arch_list[@]}"; do
        go env -w GOOS=$os 2> /dev/null
        go env -w GOARCH=$arch 2> /dev/null

        if [[ $? -eq 0 ]]; then
            echo Building for $os / $arch
            go build -o "${build_path}/${project_name}_${os}_${arch}${extension}" $main_file
        fi
    done
done

go env -u GOOS
go env -u GOARCH
