# verus_app_backend

Core platform logic, such as API endpoints that our customers would access to upload applicants, the connection to the Compliance vendors, and the business logic to provide responses.

### Development on Docker

Download repo: https://github.com/rachel-lawrie/verus_app_backend
Delete go.sum
Run `go mod tidy` to get rid of errors.

Create a .env file in the repository using the .env.example file as a template. Copy db username and password from shared/docker-compose-shared.dev.yml. Add your own AWS credentials. For testing in development, you can put a placeholder value for the webhook secret key.

Run the following to start the server:
```bash
docker-compose -f docker-compose.dev.yml build --no-cache  
docker-compose -f docker-compose.dev.yml up --remove-orphans
```

- IDE in container:

Run:
```bash
docker exec -it verusinc-app bash
```
replace "verusinc-app" with different name if needed. 

- Identify database container name
```bash
docker ps
```

- **Database**
A mongodb container is launched as part of the docker set up.

To access database commands in IDE:

Run the following (in this example it is mongodb-container (replace with relevant name if needed))
```bash
docker exec -it mongodb-container mongosh -u admin -p adminpassword
```

- Identify database container name.
```bash
docker ps
```
