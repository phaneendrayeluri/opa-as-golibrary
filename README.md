# opa-as-golibrary
A microservice written in go secured with opa

Run the application - `go run main.go`

The application has two api endpoints
1. /fetch (policy allows only user "sai" to GET to the api endpoint)
2. /udpate (policy allows only user "admin" to POST to the api endpoint)

*see the policies file

### Example Requests
`curl -X GET 'http://localhost:8080/fetch?user=sai'` <- 200 OK

`curl -X GET 'http://localhost:8080/fetch?user=admim'` <- 401 UnAuthorized

`curl -X POST 'http://localhost:8080/update?user=sai'` <- 401 UnAuthorized

`curl -X POST 'http://localhost:8080/update?user=admin'` <- 204 NoContent