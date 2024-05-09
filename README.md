# Bank REST API

A simple bank REST API backend tailored for learning and practicing Go, featuring functionalities like user sign-in, JWT authentication, and PostgreSQL database integration.

### Endpoints

    POST: /login - Login user given email and password, respond with account ID and JWT token
    GET: /account - Lists all accounts
    POST: /account - Create an account
    GET: /account/{id} - Get account with JWT Token for the header