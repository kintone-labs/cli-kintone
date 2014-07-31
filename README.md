kintone-ci
==========

You may fix github.com/djimenez/iconv-go/converter.go as follows.

\#cgo windows LDFLAGS: <PATH_TO_LIB>/libiconv.a -liconv

# Usage
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

# Examples

Export all columns from an app.
$ kintone-ci -a <APP_ID> -d <DOMAIN_NAME> -t <API_KEY>

Export the specified columns to csv as Shif-JIS.
$ kintone-ci -a <APP_ID> -d <DOMAIN_NAME> -e sjis -c "$id, name1, name2" -t <API_KEY> -f <OUTPUT_FILE>

If the file has $id columns, the original data will be updated. If not, New row will be inserted.
$ kintone-ci -a <APP_ID> -d <DOMAIN_NAME> -t <API_KEY> -f <INPUT_FILE>
