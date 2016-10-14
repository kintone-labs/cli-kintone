cli-kintone
==========

cli-kintone is a command line utility for kintone.

## Version

0.7

## How to Build

### Requirement

- Go 1.2 or later
- Git and Mercurial to be able to clone the packages

Getting the source code

    $ cd ${GOPATH}/src
    $ git clone https://github.com/kintone/cli-kintone.git

Install dependencies

    $ go get github.com/kintone/go-kintone
    $ go get github.com/howeyc/gopass
    $ go get golang.org/x/text/encoding

build

    $ cd ${GOPATH}/src/cli-kintone
    $ go build

## Downloads

These binaries are available for download.

- Windows
- Linux
- Mac OS X

https://github.com/kintone/cli-kintone/releases

## Usage

    -D=false: Delete records before insert. You can specify the deleting record condition by option "-q"
    -P="": Basic authentication password
    -U="": Basic authentication user name
    -a=0: App ID
    -b="": Attachment file directory
    -c="": Field names (comma separated)
    -d="": Domain name
    -e="utf-8": Character encoding: 'utf-8'(default), 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp'
    -f="": Input file path
    -g=0: Guest Space ID
    -o="csv": Output format: 'json' or 'csv'(default)
    -p="": Password
    -q="": Query string
    -t="": API token
    -u="": Login name

## Examples

Export all columns from an app.

    $ cli-kintone -a <APP_ID> -d <DOMAIN_NAME> -t <API_TOKEN>

Export the specified columns to csv file as Shif-JIS encoding.

    $ cli-kintone -a <APP_ID> -d <DOMAIN_NAME> -e sjis -c "$id, name1, name2" -t <API_TOKEN> > <OUTPUT_FILE>

If the file has $id column, the original data will be updated. If not, new row will be inserted.

    $ cli-kintone -a <APP_ID> -d <DOMAIN_NAME> -e sjis -t <API_TOKEN> -f <INPUT_FILE>

Export and download attachment files to ./download directory.

    $ cli-kintone -a <APP_ID> -d <DOMAIN_NAME> -t <API_TOKEN> -b download

Import and upload attachment files from ./upload directory.

    $ cli-kintone -a <APP_ID> -d <DOMAIN_NAME> -t <API_TOKEN> -b upload -f <INPUT_FILE>

Import and update by selecting a key to bulk update.  
The key to bulk update must be specified within the INPUT_FILE by placing an * in front of the field name,  
e.g. “update_date",“*id",“status"

    $ cli-kintone -a <APP_ID> -d <DOMAIN_NAME> -e sjis -t <API_TOKEN> -f <INPUT_FILE>



## Licence

GPL v2

## Copyright

Copyright(c) Cybozu, Inc.
