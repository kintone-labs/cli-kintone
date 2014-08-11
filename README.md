kintone-ci
==========

kintone-ci is a command line utility for kintone.

## Version

0.1

## Requirement

- Go 1.2 or later
- Git and Mercurial to be able to clone the packages

## How to Build

Getting the source code

    $ cd ${GOPATH}/src
    $ git clone https://github.com/kintone/kintone-ci.git

Install dependencies

    $ go get github.com/cybozu/go-kintone
    $ go get github.com/howeyc/gopass
    $ go get code.google.com/p/go.text/encoding

build

    $ cd ${GOPATH}/src/kintone-ci
    $ go build

## Usage

    -D=false: Delete all records before insert
    -a=0: App ID
    -c="": Field names (comma separated)
    -d="": Domain name
    -e="utf-8": Character encoding: 'utf-8'(default), 'sjis' or 'euc'
    -f="": Input file path
    -o="csv": Output format: 'json' or 'csv'(default)
    -p="": Password
    -q="": Query string
    -t="": API token
    -u="": Login name

## Examples

Export all columns from an app.

    $ kintone-ci -a <APP_ID> -d <DOMAIN_NAME> -t <API_TOKEN>

Export the specified columns to csv file as Shif-JIS encoding.

    $ kintone-ci -a <APP_ID> -d <DOMAIN_NAME> -e sjis -c "$id, name1, name2" -t <API_TOKEN> > <OUTPUT_FILE>

If the file has $id column, the original data will be updated. If not, new row will be inserted.

    $ kintone-ci -a <APP_ID> -d <DOMAIN_NAME> -e sjis -t <API_TOKEN> -f <INPUT_FILE>

## Licence

GPL v2

## Copyright

Copyright(c) Cybozu, Inc.
