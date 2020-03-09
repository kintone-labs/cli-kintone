## How to Build Mac OS X/Linux
#### Step 1: Creating folder to develop
```
mkdir -p /tmp/dev-cli-kintone/src
```
Note:  "/tmp/dev-cli-kintone" is the path to project at local, can be changed to match with the project at local of you.

#### Step 2: Creating variable environment GOPATH

```
export GOPATH=/tmp/dev-cli-kintone
```

#### Step 3: Getting cli-kintone repository
```
cd ${GOPATH}/src
git clone https://github.com/kintone/cli-kintone.git
```

#### Step 4: Install dependencies
```
cd ${GOPATH}/src/cli-kintone
go get github.com/mattn/gom
sudo ln -s $GOPATH/bin/gom /usr/local/bin/gom # Link package gom to directory "/usr/local/" to use globally
gom -production install
```

#### Step 5: Build
```
mv vendor/ src
gom build
```

## Copyright

Copyright(c) Cybozu, Inc.
