# URTC — Unity Real-Time Collaboration

A real-time collaboration plugin for Unity that lets teams push/pull project files via GitHub and collaborate with teammates through a session-based server.

## Structure

```
urtc-server/
├── frontend/   # Unity Editor plugin (C#)
└── backend/    # Go server (REST API + WebSocket)
```

## Features
- 🔐 GitHub OAuth login
- 📤 Push Unity project to GitHub
- 📥 Pull latest changes from GitHub
- 👥 Add collaborators by email
- 🔗 Join token system for collaborators
- 🔔 Real-time WebSocket notifications
- 📁 File sharing between collaborators
- 🕒 Version control & conflict detection

## Backend Setup

```bash
cd backend
cp .env.example .env
# Fill in your DATABASE_URL, GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET
go run .
```

## Frontend Setup

1. Open the `frontend/` folder as a Unity project (Unity 2021.3+)
2. Go to **Window → URTC Panel**
3. Set your Server URL and login with GitHub

## How to Collaborate

**Owner:**
1. Login with GitHub → Start Collaboration
2. Add collaborator by email → Copy Join Token

**Collaborator:**
1. Open URTC Panel → Collaborator tab
2. Enter your email + Join Token → Join
3. Pull latest changes from GitHub

## Backend Requirements
- Go 1.21+
- PostgreSQL database
- GitHub OAuth App
