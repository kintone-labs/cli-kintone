cli-kintone
==========

cli-kintone is a command line utility for exporting and importing kintone App data.

## Version

0.9.0

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
    $ go get github.com/jessevdk/go-flags

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
    Usage:
        cli-kintone.darwin.amd64 [OPTIONS]
    Application Options:
        -d, --domain=         Domain name
        -u, --username=       Login name
        -p, --password=       Password
        -U, --basic-username= Basic authentication user name
        -P, --basic-password= Basic authentication password
        -t, --api-token=      API token
        -o, --output-format=  Output format: 'json' or 'csv' (default: csv)
        -q, --query=          Query string
        -a, --app-id=         App ID (default: 0)
        -c, --columns=        Field names (comma separated)
        -f, --input-file=     Input file path
        -D, --delete-all      Delete all records before inserting
        -e, --encoding=       Character encoding: 'utf-8', 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp' (default: utf-8)
        -g, --guest-space-id= Guest Space ID (default: 0)
        -b, --attachment-dir= Attachment file directory
        -l, --line=           The position index of data in the input file (default: 1)
            --import          Force import
            --export          Force export

    Help Options:
        -h, --help            Show this help message

## Examples

### Export all columns from an app

    $ cli-kintone -a <APP_ID> -d <FQDN> -t <API_TOKEN>

### Export the specified columns to csv file as Shif-JIS encoding

    $ cli-kintone -a <APP_ID> -d <FQDN> -e sjis -c "$id, name1, name2" -t <API_TOKEN> > <OUTPUT_FILE>

### Import specified file into an App

    $ cli-kintone --import -a <APP_ID> -d <FQDN> -e sjis -t <API_TOKEN> -f <INPUT_FILE>

Records are updated and/or added if the import file contains either an $id column (that represents the Record Number field), or a column representing a key field (denoted with a * symbol before the field code name, such as "\*mykeyfield").  

If the value in the $id (or key field) column matches a record number value, that record will be updated.  
If the value in the $id (or key field) column is empty, a new record will be added.  
If the value in the $id (or key field) column does not match with any record number values, the import process will stop, and an error will occur.  
If an $id (or key field) column does not exist in the file, new records will be added, and no records will be updated.

### Export and download attachment files to ./mydownloads directory

    $ cli-kintone -a <APP_ID> -d <FQDN> -t <API_TOKEN> -b mydownloads

### Import and upload attachment files from ./myuploads directory

    $ cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN> -b myuploads -f <INPUT_FILE>

### Import and update by selecting a key to bulk update
The key to bulk update must be specified within the INPUT_FILE by placing an * in front of the field code name,  
e.g. “update_date",“*id",“status".

    $ cli-kintone --import -a <APP_ID> -d <FQDN> -e sjis -t <API_TOKEN> -f <INPUT_FILE>

### Import CSV from line 25 of the input file

     $ cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN> -f <INPUT_FILE> -l 25

### Import from standard input (stdin)

     $ printf "name,age\nJohn,37\nJane,29" | cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN>

## Documents for Basic Usage
English: https://developer.kintone.io/hc/en-us/articles/115002614853  
Japanese: https://developer.cybozu.io/hc/ja/articles/202957070

## Restriction
* The limit of file upload size is 10 MB.

## License

GPL v2

## Copyright

Copyright(c) Cybozu, Inc.
