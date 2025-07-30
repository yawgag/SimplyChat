# SimplyChat
A simple, scalable real-time messaging application built on the Go architecture, WebSocket, and microservices.

## Features
- **Real-time Messaging:** Instant message delivery using WebSocket protocol.
- **User Authentication:** Secure JWT-based authentication with refresh tokens.
- **Microservices Architecture:** Decoupled services for scalability and maintainability.

## Quick start
The application runs on `localhost:8080`.

**Requirements:**
- Docker
- Docker Compose

**Commands:**

```bash
make build  # Build Docker images
make up     # Start the application
make down   # Stop the application
```

## Architecture
The system is built using a microservices architecture:
```
Frontend (HTML/CSS/JS) - API Gateway - Message Service
                              |
                          Auth Service
```
## Video

[Messanger video](https://drive.google.com/file/d/1Bp6PKxC69SCokWj93GURFGulpGvfEXjr/view?usp=drive_link)
