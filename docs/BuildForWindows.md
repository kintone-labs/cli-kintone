## How to Build Windows
#### Step 1: Creating folder to develop
```
mkdir -p c:\tmp\dev-cli-kintone\src
```
Note: "c:\tmp\dev-cli-kintone" is the path to project at local, can be changed to match with the project at local of you.

#### Step 2: Creating variable environment GOPATH

```
set GOPATH=c:\tmp\dev-cli-kintone
```

#### Step 3: Getting cli-kintone repository
```
cd %GOPATH%\src
git clone https://github.com/kintone/cli-kintone.git
```

#### Step 4: Install dependencies
```
cd %GOPATH%\src\cli-kintone
go get github.com/mattn/gom
..\..\bin\gom.exe -production install
```

#### Step 5: Build
```
..\..\bin\gom.exe build
```

## Copyright

Copyright(c) Cybozu, Inc.
