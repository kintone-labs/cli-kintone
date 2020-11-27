cli-kintone
==========

cli-kintone is a command line utility for exporting and importing kintone App data.

## Version

0.11.2

## Downloads

These binaries are available for download.

- Windows
- Linux
- Mac OS X

https://github.com/kintone/cli-kintone/releases

## Usage
```text
    Usage:
        cli-kintone [OPTIONS]

    Application Options:
            --import  Import data from stdin. If "-f" is also specified, data is imported from the file instead
            --export  Export kintone data to stdout
        -d=           Domain name (specify the FQDN)
        -a=           App ID (default: 0)
        -u=           User's log in name
        -p=           User's password
        -t=           API token
        -g=           Guest Space ID (default: 0)
        -o=           Output format. Specify either 'json' or 'csv' (default: csv)
        -e=           Character encoding (default: utf-8).
                        Only support the encoding below both field code and data itself:  
                        'utf-8', 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or'euc-jp', 'gbk' or 'big5'
        -U=           Basic authentication user name
        -P=           Basic authentication password
        -q=           Query string
        -c=           Fields to export (comma separated). Specify the field code name
        -f=           Input file path
        -b=           Attachment file directory
        -D            Delete records before insert. You can specify the deleting record condition by option "-q"
        -l=           Position index of data in the input file (default: 1)
        -v, --version Version of cli-kintone

    Help Options:
        -h, --help    Show this help message
```
## Examples
Note: 
* If you use Windows device, please specify cli-kintone.exe
* Please set the PATH to cli-kintone to match your local environment beforehand.

### Export all columns from an app
```
cli-kintone --export -a <APP_ID> -d <FQDN> -t <API_TOKEN>
```
### Export the specified columns to csv file as Shif-JIS encoding
```
cli-kintone --export -a <APP_ID> -d <FQDN> -e sjis -c "$id, name1, name2" -t <API_TOKEN> > <OUTPUT_FILE>
```
### Import specified file into an App
```
cli-kintone --import -a <APP_ID> -d <FQDN> -e sjis -t <API_TOKEN> -f <INPUT_FILE>
```
Records are updated and/or added if the import file contains either an $id column (that represents the Record Number field), or a column representing a key field (denoted with a * symbol before the field code name, such as "\*mykeyfield").  

If the value in the $id (or key field) column matches a record number value, that record will be updated.  
If the value in the $id (or key field) column is empty, a new record will be added.  
If the value in the $id (or key field) column does not match with any record number values, the import process will stop, and an error will occur.  
If an $id (or key field) column does not exist in the file, new records will be added, and no records will be updated.

### Export and download attachment files to ./mydownloads directory
```
cli-kintone --export -a <APP_ID> -d <FQDN> -t <API_TOKEN> -b mydownloads
```
### Import and upload attachment files from ./myuploads directory
```
cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN> -b myuploads -f <INPUT_FILE>
```
### Import and update by selecting a key to bulk update
The key to bulk update must be specified within the INPUT_FILE by placing an * in front of the field code name,  
e.g. “update_date",“*id",“status".

```
cli-kintone --import -a <APP_ID> -d <FQDN> -e sjis -t <API_TOKEN> -f <INPUT_FILE>
```
### Import CSV from line 25 of the input file
```
cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN> -f <INPUT_FILE> -l 25
```
### Import from standard input (stdin)
```
printf "name,age\nJohn,37\nJane,29" | cli-kintone --import -a <APP_ID> -d <FQDN> -t <API_TOKEN>
```

## Restriction
* The limit of the file upload size is 10 MB.
* Client certificates cannot be used with cli-kintone.
* The following record data cannot be retrieved: Category, Status, Field group.
* The following fields cannot be retrieved if they are set inside a Field group: Record number, Created by, Created datetime, Updated by, Updated datetime, Blank space, Label, Border.

## Restriction of Encode/Decode
* Windows command prompt may not display characters correctly like "譁�蟄怜喧縺�".  
  This is due to compatibility issues between Chinese & Japanese characters and the Windows command prompt.
  * Chinese (Traditional/Simplified): Display wrong even if exporting with gbk or big5 encoding.
  * Japanese: Display wrong even if exporting with sjis or euc-jp encoding.
  
  In this case, display the data by specifying utf-8 encoding like below:
  ```
  cli-kintone --export -a <APP_ID> -d <FQDN> -e utf-8
  ```
  *This issue only occurs when displaying data on Windows command prompt. Data import/export with other means work fine with gbk, big5, sjis and euc-jp encoding.

## Documents for Basic Usage
English: https://developer.kintone.io/hc/en-us/articles/115002614853  
Japanese: https://developer.cybozu.io/hc/ja/articles/202957070

## How to Build

Requirement

- Go 1.15.5
- Git and Mercurial to be able to clone the packages

[Mac OS X/Linux](./docs/BuildForMacLinux.md)

[Windows](./docs/BuildForWindows.md)

## License

GPL v2

## Copyright

Copyright(c) Cybozu, Inc.
