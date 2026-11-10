# UB Lost and Found System

A web-based Lost and Found Management System designed for the University of Belize. The system helps students, faculty, and staff search for missing belongings while allowing authorized office staff to manage found items and verify claims.

## Features

### Public Features
- Search for found items.
- Filter items by category, date, and location.
- View available item information.
- Submit a claim for a found item.
- No account required for public item searches or claim submissions.

### Staff Features
- Secure staff login and logout.
- Add and manage found item records.
- Review submitted claims.
- Approve or reject claims with verification notes.
- Automatically update item statuses.
- View inventory statistics and category reports.

### Status Management
- **Unclaimed:** An item is available to be claimed.
- **Pending Verification:** Someone has submitted a claim that needs staff review.
- **Claimed:** Staff have approved a claim.

## Technologies Used

- HTML
- CSS
- JavaScript
- Go (Golang)
- MySQL
- GitHub

## Project Structure

```text
UB-Lost-and-Found/
├── database/
│   └── schema.sql
├── images/
├── public/
│   ├── index.html
│   ├── css/
│   │   └── style.css
│   ├── js/
│   │   └── app.js
│   └── images/
│       └── ub-img.jpeg
├── tools/
│   └── hashpass.go
├── .gitignore
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## Requirements

Before running the application, make sure you have:

- Go installed.
- MySQL Server installed and running.
- Git (optional, if cloning the repository).

## Installation and Setup

### 1. Download the Project

Clone the repository:

```bash
git clone https://github.com/Ojim12/UB-Lost-and-Found.git
cd UB-Lost-and-Found
```

You can also download the repository as a ZIP file from GitHub.

### 2. Set Up the Database

Import the database schema from the project folder:

```bash
mysql -u root -p < database/schema.sql
```

The database used by the application is named `ub_lost_found`.

### 3. Configure Database Access

Create a separate MySQL application account instead of using the root account.

In MySQL, create an account with a password you choose and grant it access:

```sql
CREATE USER 'ub_web'@'localhost'
IDENTIFIED BY 'YOUR_OWN_STRONG_PASSWORD';

GRANT SELECT, INSERT, UPDATE, DELETE
ON ub_lost_found.*
TO 'ub_web'@'localhost';
```

### 4. Set Environment Variables

In Git Bash, enter:

```bash
export DB_USER='ub_web'
read -s -p "Enter your MySQL application password: " DB_PASSWORD
echo
export DB_PASSWORD
```

The password is entered privately and is not stored in the source code.

### 5. Install Go Dependencies

```bash
go mod download
```

### 6. Run the Application

```bash
go run .
```

Open your browser and visit:

http://localhost:8080

## Staff Account Setup

The project includes a password-hashing utility in `tools/hashpass.go`.

It uses the `STAFF_PASSWORD` environment variable to generate a bcrypt password hash. Use the generated hash when creating a staff account in the database.

Never publish actual staff passwords or password hashes in this repository.

## Important Notes

- Public visitors can search for items and submit claims without logging in.
- Staff authentication is required for protected management features.
- The application requires a running MySQL database.
- The current session storage is in memory, so staff sessions reset when the Go server restarts.
- The application is currently intended for local development and demonstration. Additional security and deployment configuration are needed before production use.

## Project Purpose

This project was developed as part of a Systems Analysis and Design project at the University of Belize. Its purpose is to improve how lost and found items are recorded, searched, managed, and returned to their rightful owners.

## Repository

https://github.com/Ojim12/UB-Lost-and-Found
