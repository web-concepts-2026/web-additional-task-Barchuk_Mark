# Basic HTTP Server 

```bash
go run main.go
```

## GET 
Open `http://localhost:8080/page`

## GET /page?id=5

Open `http://localhost:8080/page?id=5`

## POST /echo


```bash
curl -i -X POST http://localhost:8080/echo \
  -H 'Content-Type: application/json' \
  -d '{"message":"pls put me 4 points"}'
```
