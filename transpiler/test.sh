#!/bin/sh
# Gofasta Transpiler Installation Script

rm -rf tmp
rm -rf dist
./build.sh
./dist/gofasta -i examples -o tmp

