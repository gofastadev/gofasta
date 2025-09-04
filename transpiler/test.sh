#!/bin/bash
# Gofasta Transpiler Installation Script

rm -rf dist
./build.sh
./dist/gofasta -i examples -o tmp
