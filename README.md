# Accounting sample with CQRS/ES
### Build
`docker-compose -f docker-compose.build.yml build`
### Run
`docker-compose up`
### Add Transaction
```http
POST /transaction HTTP/1.1
Host: localhost:7000
Content-Type: application/json

{
    "asdf": 1234,
    "amount": 100,
    "occurred": "2006-01-02T15:04:05.999999999Z"
}
```
### Delete Transaction
```http
DELETE /transaction/9508bede-08ea-11ea-a0fd-00ff2fa3c8cb HTTP/1.1
Host: localhost:7000
```

### List Transactions
```http
GET /transaction HTTP/1.1
Host: localhost:7001
```

### Get Balance
```http
GET /balance HTTP/1.1
Host: localhost:7001
```

### List Transactions (snapshot)
```http
GET /transaction?snapshot=2019-11-17T03:32:03.6481079Z HTTP/1.1
Host: localhost:7001
```

### Get Balance (snapshot)
```http
GET /balance?snapshot=2019-11-17T03:32:03.6481079Z HTTP/1.1
Host: localhost:7001
```

### Event Store History
```http
GET /history HTTP/1.1
Host: localhost:8000
```
