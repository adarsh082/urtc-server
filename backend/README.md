# RTC Server — **REAL TIME COLLABORATION TOOL FOR GAME DEVELOPMENT SOFTWARES**

> **RTC** (Real-Time Collaboration Server) is a Go-based backend providing real-time collaboration, version control, file sharing, and GitHub OAuth for Unity teams.This is a real time collaboration tool project created especially for game development engines like godot and unity, this project aims to build a real time connection for live sharing of assets and scripts, while maintaining minimal background operations. 


---

## 🔐 Authentication & Headers

Almost all endpoints (except public ones and GitHub login) are **Protected**.
They require a valid Session ID, obtained from `/github/callback` after login.

### Required Headers for Protected Routes

| Header | Value | Description |
|---|---|---|
| `X-Session-ID` | `<session_id>` | **Required**. Proves authentication. |
| `Content-Type` | `application/json` | Required for `POST`, `PUT`, `DELETE`. |
| `X-User-ID` | `<uuid>` | Optional. Injects user ID context. |
| `X-Project-ID`| `<uuid>` | Optional. Injects project ID context. |

Alternatively, the `X-Session-ID` can be provided via a `session_id` cookie.

---

## 🌐 1. Public Routes

No authentication or headers required.

### 1.1 Health Check
**GET** `/` or `/health`
- **Output:** `200 OK` (String: `"URTC Server is running!"` or `"OK"`)

### 1.2 Get Total Users Count
**GET** `/db/users-count`
- **Output:** `200 OK` (String: `"No of Users : <count>"`)

### 1.3 List All Users
**GET** `/db/users`
- **Output:** Plaintext list of users.

### 1.4 Get User by Username / Email
**GET** `/db/users/{username}`
**GET** `/db/user/{email}`
- **Path Params:** `username` or `email` (string)
- **Output:** `200 OK` (String: `"ID : <id> , Username : <name>, ... "`)
- **Error:** `400 Bad Request` if not found.

---

## 🔑 2. GitHub OAuth Authentication

### 2.1 Initiate Login
**GET** `/github/login`
- **Action:** Redirects the user to the GitHub OAuth authorization page.

### 2.2 GitHub Callback
**GET** `/github/callback?code={auth_code}`
- **Query Params:** `code` (string, from GitHub redirect)
- **Action:** Exchanges code for an access token, upserts the user, and creates a session.
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "Login successful",
    "session_id": "base64_url_encoded_string",
    "user": {
      "id": "uuid",
      "username": "developer1",
      "email": "dev@example.com"
    }
  }
  ```
- **Important:** Returns the `X-Session-ID` in the response headers.

### 2.3 Logout
**POST** `/github/logout`
- **Headers:** `X-Session-ID`
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "Logged out successfully"
  }
  ```

---

## 📦 3. Project Management (Protected)

Requires `X-Session-ID`.

### 3.1 Get Projects Count
**GET** `/api/projects-count/{owner}`
- **Path Params:** `owner` (user ID UUID)
- **Output (`200 OK`):** Plaintext count of projects.

### 3.2 List Owner Projects
**GET** `/api/projects/{owner}`
- **Path Params:** `owner` (user ID UUID)
- **Output (`200 OK`):** Plaintext list of projects (JSON encoded in loop).

### 3.3 Create Project / Start Collaboration
**POST** `/api/push/manual`
**POST** `/api/start-collaboration`
- **Description:** Creates a GitHub repository for the user and saves it locally.
- **Input Body:**
  ```json
  {
    "user_email": "dev@example.com",
    "project_name": "MyUnityGame"
  }
  ```
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "Collaboration started successfully",
    "project_id": "uuid",
    "repo_url": "https://github.com/developer1/MyUnityGame",
    "github_token": "ghp_xxxx1234"
  }
  ```
- **Errors:** `400 Bad Request` if project name already exists.

---

## 🤝 4. Collaboration System (Protected)

Requires `X-Session-ID`.

### 4.1 Request Collaboration (Owner invites Collaborator)
**POST** `/api/collab/request`
- **Description:** Sends an invite to a GitHub-authenticated user.
- **Input Body:**
  ```json
  {
    "owner_email": "owner@example.com",
    "collaborator_email": "collab@example.com",
    "project_id": "uuid"
  }
  ```
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "Collaboration request sent successfully",
    "collab_id": "uuid",
    "status": "pending",
    "notification": {
      "collab_id": "uuid",
      "project_id": "uuid",
      "project_name": "MyGame",
      "project_owner": "owner",
      "collaborator_email": "collab@example.com",
      "collaborator_name": "collabuser",
      "status": "pending",
      "created_at": "2024-01-01 12:00:00"
    }
  }
  ```

### 4.2 Approve/Reject Collaboration (By Collaborator)
**POST** `/api/collab/approve`
- **Description:** The invited collaborator confirms or rejects the invite.
- **Input Body:**
  ```json
  {
    "collab_id": "uuid",
    "status": "approved" 
  }
  ```
  *(Status must be `"approved"` or `"rejected"`)*
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "Collaboration request approved",
    "collab_id": "uuid",
    "status": "approved"
  }
  ```

### 4.3 Join Collaboration (Unity Plugin Shortcut)
**POST** `/api/join-collaboration`
- **Description:** Unity plugin quick-join using the `collab_id` as a token. Automatically approves.
- **Input Body:**
  ```json
  {
    "user_email": "collab@example.com",
    "token": "collab_id_uuid_here"
  }
  ```
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "Joined collaboration successfully",
    "project_id": "uuid",
    "repo_url": "https://github.com/owner/MyGame",
    "github_token": "ghp_xxxx123"
  }
  ```

### 4.4 Get Project Pending Collaborations (For Owner)
**GET** `/api/collab/pending?project_name={name}`
- **Query Params:** `project_name`
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "project_name": "MyGame",
    "project_id": "uuid",
    "owner": "owner_user",
    "requests": [
      {
        "collab_id": "uuid",
        "collaborator_id": "uuid",
        "collaborator_name": "collab",
        "collaborator_email": "collab@example.com",
        "project_id": "uuid",
        "project_name": "MyGame",
        "status": "pending",
        "requested_at": "2024-01-01 12:00:00"
      }
    ],
    "total": 1
  }
  ```

### 4.5 List All Collaborators for Project
**GET** `/api/collab/project?project_id={uuid}`
- **Query Params:** `project_id`
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "project_id": "uuid",
    "collaborators": [
      {
        "id": "collab_uuid",
        "user_id": "user_uuid",
        "project_id": "project_uuid",
        "status": "approved",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
  ```

### 4.6 List Pending Invites Sent to User
**GET** `/api/collab/user/requests?user_email={email}`
- **Query Params:** `user_email`
- **Output (`200 OK`):** JSON list of pending invites where the user is the collaborator.

### 4.7 Remove Collaborator
**DELETE** `/api/collab/remove/{collab_id}`
- **Path Params:** `collab_id`
- **Output (`200 OK`):** `{ "success": true, "message": "Collaborator removed successfully" }`

---

## 📁 5. File Sharing (Protected via WebSocket Push)

Requires `X-Session-ID`. All outputs are HTTP `200 OK` confirmation, but the actual payload is pushed to the recipient's WebSocket.

### 5.1 Share File
**POST** `/api/share/file`
- **Input Body:**
  ```json
  {
    "sender_email": "sender@example.com",
    "recipient_email": "recipient@example.com",
    "project_id": "uuid",
    "file_name": "Player.cs",
    "file_content": "base64_encoded_string",
    "file_type": "script",
    "message": "Here is the player script"
  }
  ```
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "message": "File shared successfully",
    "file_name": "Player.cs",
    "sent_to": "recipient_username",
    "timestamp": "2024-01-01T00:00:00Z"
  }
  ```

### 5.2 Share Code Snippet
**POST** `/api/share/code`
- **Input Body:**
  ```json
  {
    "sender_email": "sender@example.com",
    "recipient_email": "recipient@example.com",
    "project_id": "uuid",
    "file_name": "Game.cs",
    "code": "void Update() {}",
    "language": "csharp",
    "line_number": 42,
    "message": "Fix this line"
  }
  ```
- **Output (`200 OK`):** Similar to Share File.

### 5.3 Share Bulk Files
**POST** `/api/share/bulk`
- **Input Body:**
  ```json
  {
    "sender_email": "sender",
    "recipient_email": "recipient",
    "project_id": "uuid",
    "message": "New assets",
    "files": [
      {
        "file_name": "1.png",
        "file_content": "base64",
        "file_type": "asset"
      }
    ]
  }
  ```

### 5.4 Get Shareable Collaborators
**GET** `/api/share/collaborators?project_id={uuid}`
- **Output:** Returns all `approved` collaborators with their real-time `is_online` status derived from active WebSockets.

---

## 📝 6. Version Control (Protected)

Requires `X-Session-ID`.

### 6.1 Commit File
**POST** `/api/version/commit`
- **Input Body:**
  ```json
  {
    "project_id": "uuid",
    "user_email": "user@example.com",
    "file_path": "Assets/Scripts/GameManager.cs",
    "file_name": "GameManager.cs",
    "file_type": "script",
    "content": "raw_or_base64_content",
    "file_hash": "sha256_hex_hash",
    "file_size": 1024,
    "commit_message": "Added game loop",
    "base_version": 4
  }
  ```
- **Output - Success (`200 OK`):**
  ```json
  {
    "success": true,
    "version_id": "uuid",
    "version": 5,
    "file_path": "Assets/Scripts/GameManager.cs",
    "commit_msg": "Added game loop",
    "has_conflict": false
  }
  ```
- **Output - Conflict (`409 Conflict`):**
  ```json
  {
    "success": false,
    "conflict": true,
    "conflict_id": "uuid",
    "message": "Conflict detected. Please resolve before committing.",
    "base_version": 4,
    "latest_version": 5
  }
  ```

### 6.2 Get File History
**GET** `/api/version/history?project_id={uuid}&file_path={path}`
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "project_id": "uuid",
    "file_path": "path/to/file.cs",
    "versions": [
      {
        "id": "uuid",
        "version": 5,
        "user_id": "uuid",
        "username": "developer1",
        "commit_message": "Added game loop",
        "file_hash": "sha256hex",
        "file_size": 1024,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
  ```

### 6.3 Get Project Conflicts
**GET** `/api/version/conflicts?project_id={uuid}`
- **Output (`200 OK`):** Array of conflicts with `status: "pending"`. Include `local_content` and `remote_content` for diff generation.

### 6.4 Resolve Conflict
**POST** `/api/version/resolve`
- **Input Body:**
  ```json
  {
    "conflict_id": "uuid",
    "resolved_by_email": "user@example.com",
    "resolved_content": "final_merged_content",
    "commit_message": "Merge resolution"
  }
  ```
- **Output (`200 OK`):** `{ "success": true, "message": "Conflict resolved successfully" }`

---

## 📊 7. Activity Tracking (Protected)

Requires `X-Session-ID`.

### 7.1 Get User / Project / Team Activities
**GET** `/api/activity/user?user_email={email}&limit=50`
**GET** `/api/activity/project?project_id={uuid}&limit=50`
**GET** `/api/activity/team?project_id={uuid}&limit=100`

- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "activities": [
      {
        "id": "uuid",
        "user_id": "uuid",
        "project_id": "uuid",
        "action": "file_commit",
        "description": "Committed Player.cs",
        "metadata": {
          "file_path": "Player.cs",
          "version": 5
        },
        "ip_address": "127.0.0.1",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
  ```

---

## 🔌 8. WebSocket Events (Protected)

Requires `user_id` query param. Used for real-time pushing.

### 8.1 Connect
**GET** `/api/ws?user_id={uuid}`
- Upgrades to WS protocol. Must be a valid UUID. Keep alive with ping/pong every 30s.

### 8.2 Get Online Users
**GET** `/api/ws/online-users`
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "online_users": ["uuid1", "uuid2"],
    "total_online": 2
  }
  ```

### 8.3 Check User Online Status
**GET** `/api/ws/user-status?user_id={uuid}`
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "user_id": "uuid",
    "is_online": true
  }
  ```

---

## 👮 9. Admin Routes

### 9.1 Get User GitHub Token
**GET** `/admin/token/{super_user_key}/{username}`
- **Path Params:**
  - `super_user_key`: Must exactly match the `SECRET_KEY` env var.
  - `username`: GitHub username.
- **Output (`200 OK`):**
  ```json
  {
    "success": true,
    "id": "token_id_uuid",
    "username": "developer1",
    "github_token": "ghp_xxxxxxxxxxxx",
    "user_id": "uuid",
    "created_at": "2024-01-01T00:00:00Z"
  }
  ```
- **Error:** `401 Unauthorized` if key is wrong.
