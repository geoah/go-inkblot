curl -H "Content-Type: application/json" -X POST -d '{"hostname":"afternoon-atoll-8685.herokuapp.com"}' http://geoah-ink.herokuapp.com/identities
curl -H "Content-Type: application/json" -X POST -d '{"hostname":"afternoon-atoll-8685.herokuapp.com"}' http://geoah-ink.herokuapp.com/identities

curl -H "Content-Type: application/json" -X POST -d '{"schema": "dummy", "data":"test string"}' http://localhost:5000/instances

curl -H "Content-Type: application/json" -X GET http://localhost:8000

curl -H "Content-Type:application/json" -X POST -d '{"username": "user", "password": "user"}' http://localhost:8000/login
curl -H "Content-Type: application/json" -X GET http://localhost:8000/identities
curl -H "Content-Type: application/json" -X GET --header "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0MzEyMDU5MzEsImlkIjoidXNlciIsIm9yaWdfaWF0IjoxNDMxMjAyMzMxfQ.mn5Kx7LOUc5u-dFzatnyqn--iw-ngd2qlShlzTHlURk" http://localhost:8000
