# Short URL
Simple URL shortener you can deploy using docker.

## Example
```
$ curl --data '{"url": "https://example.com/"}' localhost:8090/api/shorten
{"url":"au8nuvPueQQ6IjaKtD9Ih"}

$ curl --data '{"url": "au8nuvPueQQ6IjaKtD9Ih"}' localhost:8090/api/resolve
{"url":"https://example.com/"}
```

Simple web interface is by default available at localhost:8080

## Building
```
./build/all.sh
```

## Testing
```
./test/integration.sh
```
