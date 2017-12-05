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
    $ go get gopkg.in/yaml.v2

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

    -d = "" : Domain name. Specify the FQDN.
    -a = 0 : App ID.
    -u = "" : User's log in name
    -p = "" : User's password.
    -t = "" : API token.     
    -g = 0 : Guest Space ID.
    -o = "csv" : Output format. Specify either 'json' or 'csv'(default).  
    -e = "utf-8" : Character encoding. Specify one of the following -> 'utf-8'(default), 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp'.
    -U = "" : Basic authentication user name.
    -P = "" : Basic authentication password.         
    -q = "" : Query string. 
    -c = "" : Fields to export (comma separated). Specify the field code name.
    -f = "" : Input file path.
    -b = "" : Attachment file directory.
    -D = false : Delete records before insert. You can specify the deleting record condition by option "-q".
    -l = 1 : Position index of data in the input file. Default is 1.
    --import : Import data from stdin. If "-f" is also specified, data is imported from the file instead.
    --export : Export kintone data to stdout.
    
## Examples

Export all columns from an app.

    $ cli-kintone -a <APP_ID> -d <FQDN> -t <API_TOKEN>

Export the specified columns to csv file as Shif-JIS encoding.

    $ cli-kintone -a <APP_ID> -d <FQDN> -e sjis -c "$id, name1, name2" -t <API_TOKEN> > <OUTPUT_FILE>

If the file has an $id column, the original data will be updated. If not, new row will be inserted.

    $ cli-kintone -a <APP_ID> -d <FQDN> -e sjis -t <API_TOKEN> -f <INPUT_FILE>

Export and download attachment files to ./download directory.

    $ cli-kintone -a <APP_ID> -d <FQDN> -t <API_TOKEN> -b download

Import and upload attachment files from ./upload directory.

    $ cli-kintone -a <APP_ID> -d <FQDN> -t <API_TOKEN> -b upload -f <INPUT_FILE>

Import and update by selecting a key to bulk update.  
The key to bulk update must be specified within the INPUT_FILE by placing an * in front of the field code name,  
e.g. “update_date",“*id",“status"

    $ cli-kintone -a <APP_ID> -d <FQDN> -e sjis -t <API_TOKEN> -f <INPUT_FILE>

Import CSV from line 25 of the input file.

     $ cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN> -f <INPUT_FILE> -l 25

Import from standard input (stdin).

     $ cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN>

## Documents for Basic Usage
English: https://developer.kintone.io/hc/en-us/articles/115002614853  
Japanese: https://developer.cybozu.io/hc/ja/articles/202957070

## Restriction
* The limit of file upload size is 10 MB.

## Licence

GPL v2

## Copyright

Copyright(c) Cybozu, Inc.
