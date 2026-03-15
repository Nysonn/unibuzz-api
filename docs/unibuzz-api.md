# UniBuzz API

**Frontend Engineering Guide** | Version: March 2026

**Base URL:** `https://unibuzz-api.onrender.com`

All request and response bodies use JSON. Authenticated endpoints require an `Authorization: Bearer <access_token>` header.

---

## Endpoints at a Glance

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | /health | No | Health check |
| GET | /metrics | No | Prometheus metrics |
| POST | /auth/register | No | Register a new user |
| POST | /auth/login | No | Login and get tokens |
| GET | /api/feed | Yes | Get video feed |
| GET | /api/search | No | Search videos by hashtag or username |
| POST | /api/videos/upload | Yes | Upload a video |
| GET | /api/videos/:video_id/status | Yes | Get video processing status |
| POST | /api/videos/:video_id/comments | Yes | Add a comment |
| GET | /api/videos/:video_id/comments | No | Get comments for a video |
| PUT | /api/comments/:comment_id | Yes | Update a comment |
| DELETE | /api/comments/:comment_id | Yes | Delete a comment |
| POST | /api/videos/:video_id/vote | Yes | Vote on a video |
| GET | /api/videos/:video_id/votes | No | Get vote counts |
| POST | /api/videos/:video_id/report | Yes | Report a video |
| GET | /admin/reports | Admin | Get all reports |
| DELETE | /admin/videos/:video_id | Admin | Delete a video |
| GET | /admin/users | Admin | List all users |
| GET | /admin/users/:user_id | Admin | Get user by ID |
| POST | /admin/users/:user_id/suspend | Admin | Suspend a user |
| POST | /admin/users/:user_id/unsuspend | Admin | Unsuspend a user |
| POST | /admin/users/:user_id/ban | Admin | Ban a user |
| DELETE | /admin/users/:user_id | Admin | Delete a user |

---

## 1. Health

### GET /health

Returns server health status. No authentication required.

**Response `200`**
```json
{ "status": "ok" }
```

### GET /metrics

Returns Prometheus-format metrics. No authentication required. Intended for DevOps dashboards, not the frontend app.

---

## 2. Authentication

After a successful login, store the `access_token` securely and attach it as a Bearer token on all authenticated requests. Use the `refresh_token` to get a new `access_token` when it expires.

### POST /auth/register

Creates a new user account. No authentication required.

**Request body**
```json
{
  "full_name": "James McGill",
  "username": "jmcgill",
  "email": "james.mcgill@university.ac.ug",
  "password": "SecureKey456!",
  "university_name": "Kyambogo University",
  "course": "Computer Science",
  "year_of_study": 4
}
```

**Response `201`**
```json
{
  "id": "04ea5569-3cd7-4649-8e35-9cee7e975c52",
  "full_name": "James McGill",
  "username": "jmcgill",
  "email": "james.mcgill@university.ac.ug",
  "university_name": "Kyambogo University",
  "course": "Computer Science",
  "year_of_study": 4,
  "is_admin": false,
  "is_suspended": false,
  "profile_photo_url": null,
  "created_at": "2026-03-11T18:11:39.129833",
  "updated_at": "2026-03-11T18:11:39.129833"
}
```

### POST /auth/login

Authenticates a user and returns JWT tokens.

**Request body**
```json
{
  "email": "info@unibuzz.com",
  "password": "admin123"
}
```

**Response `200`**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "6c269178-1308-456e-96a1-a936d24dacae"
}
```

| Field | Type | Usage |
|-------|------|-------|
| `access_token` | JWT string | `Authorization: Bearer <token>` |
| `refresh_token` | UUID string | Use to get a new `access_token` when expired |

---

## 3. Feed & Videos

### GET /api/feed

Returns the 20 most recent processed videos. **Authentication required.**

**Response `200`**
```json
[
  {
    "id": "8b6d115e-12d7-4abe-ae4a-3e6b6a6487e0",
    "caption": "",
    "user_id": "6abbea32-f879-431f-9052-e10a50369d37",
    "video_url": "https://res.cloudinary.com/.../video.mp4",
    "thumbnail_url": "https://res.cloudinary.com/.../thumbnail.jpg",
    "created_at": "2026-03-09T10:30:25.94625Z"
  }
]
```

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Use for comments, votes, reports |
| `caption` | string | May be an empty string |
| `user_id` | UUID | ID of the uploader |
| `video_url` | URL | Direct Cloudinary MP4 link |
| `thumbnail_url` | URL | Cloudinary thumbnail for preview/lazy load |
| `created_at` | ISO 8601 | UTC timestamp |

---

### POST /api/videos/upload

Queues a video for processing. **Authentication required.**

Parameters are passed as **query strings** (no request body).

| Query Param | Required | Description |
|-------------|----------|-------------|
| `input_url` | Yes | Source video URL (Cloudinary embed URLs are automatically resolved) |
| `caption` | No | Video caption text |
| `tags` | No | Comma-separated hashtags, e.g. `funny,campus,viral`. Max 10. Case-insensitive. |

**Example request**
```
POST /api/videos/upload?input_url=https://res.cloudinary.com/.../video.mp4&caption=So+funny&tags=funny,campus
```

**Response `202`**
```json
{
  "status": "pending",
  "video_id": "0c5b3415-e595-43b0-8fb8-2efc7ced069f",
  "message": "video accepted and queued for processing",
  "tags": ["funny", "campus"]
}
```

> Videos are processed asynchronously. After upload, poll `GET /api/videos/:video_id/status` until `status` is no longer `"pending"`.

---

### GET /api/videos/:video_id/status

Checks the processing status of an uploaded video. **Authentication required.**

**Response — pending**
```json
{
  "video_id": "0c5b3415-e595-43b0-8fb8-2efc7ced069f",
  "status": "pending",
  "video_url": null,
  "thumbnail_url": null
}
```

**Response — complete**
```json
{
  "video_id": "0c5b3415-e595-43b0-8fb8-2efc7ced069f",
  "status": "complete",
  "video_url": "https://res.cloudinary.com/.../video.mp4",
  "thumbnail_url": "https://res.cloudinary.com/.../thumbnail.jpg"
}
```

| Status | Meaning |
|--------|---------|
| `pending` | Still processing — `video_url` and `thumbnail_url` will be `null` |
| `complete` | Ready — `video_url` and `thumbnail_url` are populated |
| `failed` | Processing error — handle gracefully in the UI |

---

### GET /api/search

Search for videos by hashtag or username. **No authentication required.**

At least one query param must be provided. Both can be combined. Matching is partial and case-insensitive (e.g. `fun` matches `#funny`). Returns up to 50 results, newest first.

| Query Param | Required | Description |
|-------------|----------|-------------|
| `tag` | No* | Partial hashtag to search for |
| `username` | No* | Partial username to search for |

*At least one of `tag` or `username` is required.

**Example requests**
```
GET /api/search?tag=funny
GET /api/search?username=jmcgill
GET /api/search?tag=campus&username=jmcgill
```

**Response `200`**
```json
[
  {
    "id": "8b6d115e-12d7-4abe-ae4a-3e6b6a6487e0",
    "caption": "Campus life!",
    "user_id": "6abbea32-f879-431f-9052-e10a50369d37",
    "video_url": "https://res.cloudinary.com/.../video.mp4",
    "thumbnail_url": "https://res.cloudinary.com/.../thumbnail.jpg",
    "created_at": "2026-03-09T10:30:25.94625Z"
  }
]
```

Returns an empty array `[]` if no results are found.

---

## 4. Comments

### POST /api/videos/:video_id/comments

Adds a comment to a video. **Authentication required.**

**Request body**
```json
{ "comment": "so funny man" }
```

**Response `201`**
```json
{
  "comment_id": "9ddab204-a2ef-40d4-9305-d49eb7aed839",
  "message": "comment added"
}
```

---

### GET /api/videos/:video_id/comments

Returns all comments for a video. No authentication required.

**Response `200`**
```json
[
  {
    "id": "9ddab204-a2ef-40d4-9305-d49eb7aed839",
    "content": "so funny man",
    "user_id": "1e9594dc-a9c4-4f2c-846c-4d9e28d1f0d1",
    "video_id": "0c5b3415-e595-43b0-8fb8-2efc7ced069f",
    "created_at": "2026-03-11T18:24:52.521701Z",
    "updated_at": "2026-03-11T18:24:52.521701Z"
  }
]
```

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Use as `comment_id` for update/delete |
| `content` | string | Comment text |
| `user_id` | UUID | Commenter's user ID |
| `video_id` | UUID | Parent video ID |
| `updated_at` | ISO 8601 | Changes when the comment is edited |

---

### PUT /api/comments/:comment_id

Updates an existing comment. **Authentication required.** Only the comment owner can update.

**Request body**
```json
{ "comment": "Het is zo leuk" }
```

**Response `200`**
```json
{ "message": "comment updated" }
```

---

### DELETE /api/comments/:comment_id

Deletes a comment. **Authentication required.** Only the comment owner or an admin can delete.

**Response `200`**
```json
{ "message": "comment deleted" }
```

---

## 5. Votes

### POST /api/videos/:video_id/vote

Cast a vote on a video. **Authentication required.**

**Request body**
```json
{ "vote_type": 1 }
```

| `vote_type` | Meaning |
|-------------|---------|
| `1` | Upvote |
| `-1` | Downvote |

**Response `200`**
```json
{ "message": "vote recorded" }
```

---

### GET /api/videos/:video_id/votes

Get upvote and downvote totals for a video. No authentication required.

**Response `200`**
```json
{
  "upvotes": 1,
  "downvotes": 0
}
```

---

## 6. Reports

### POST /api/videos/:video_id/report

Submit a report against a video for admin review. **Authentication required.**

Users must select a reason from the predefined list. When `"other"` is selected, a `custom_reason` must also be provided.

**Request body**
```json
{
  "reason": "harassment",
  "custom_reason": ""
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `reason` | string (enum) | Yes | One of the values below |
| `custom_reason` | string | Only when `reason` is `"other"` | Max 100 characters |

**Valid `reason` values**

| Value | Description |
|-------|-------------|
| `self_harm` | Content promoting or depicting self-harm |
| `harassment` | Targeted harassment or bullying |
| `inappropriate_content` | Content that violates community guidelines |
| `spam` | Spam or misleading content |
| `other` | Other — requires `custom_reason` |

**Example — predefined reason**
```json
{ "reason": "spam" }
```

**Example — other with custom message**
```json
{
  "reason": "other",
  "custom_reason": "This video is impersonating a lecturer."
}
```

**Response `201`**
```json
{ "message": "report submitted" }
```

---

## 7. Admin Endpoints

All admin endpoints require a valid `access_token` from an account with `is_admin: true`. Regular users receive `403 Forbidden`.

### GET /admin/reports

Returns all video reports submitted by users.

**Response `200`**
```json
[
  {
    "id": "2fb7347e-6f9e-4bc8-bc72-1d8630095e98",
    "video_id": "8b6d115e-12d7-4abe-ae4a-3e6b6a6487e0",
    "reporter_id": "1e9594dc-a9c4-4f2c-846c-4d9e28d1f0d1",
    "reason": "harassment",
    "custom_reason": null,
    "resolved": false,
    "created_at": "2026-03-11T18:44:10.150577Z"
  }
]
```

`custom_reason` is `null` unless the user selected `"other"`.

---

### DELETE /admin/videos/:video_id

Permanently removes a video from the platform.

**Response `200`**
```json
{ "message": "video removed" }
```

---

### GET /admin/users

Returns a list of all registered users.

**Response `200`**
```json
[
  {
    "id": "04ea5569-3cd7-4649-8e35-9cee7e975c52",
    "full_name": "James McGill",
    "username": "jmcgill",
    "email": "james.mcgill@university.ac.ug",
    "university": "Kyambogo University",
    "course": "Computer Science",
    "year_of_study": 4,
    "is_suspended": false,
    "banned": false,
    "created_at": "2026-03-11T18:11:39.129833Z"
  }
]
```

---

### GET /admin/users/:user_id

Returns details for a single user by their UUID.

**Response `200`** — same shape as a single object from `GET /admin/users`.

---

### POST /admin/users/:user_id/suspend

Temporarily suspends a user. The user cannot log in while suspended.

**Response `200`**
```json
{ "message": "user suspended" }
```

---

### POST /admin/users/:user_id/unsuspend

Lifts a suspension and restores user access.

**Response `200`**
```json
{ "message": "user unsuspended" }
```

---

### POST /admin/users/:user_id/ban

Permanently bans a user from the platform.

**Response `200`**
```json
{ "message": "user banned" }
```

---

### DELETE /admin/users/:user_id

Permanently deletes a user account and all associated data.

**Response `200`**
```json
{ "message": "user deleted" }
```

---

## 8. Integration Notes

### Authentication flow

1. Call `POST /auth/login` with user credentials.
2. Store the `access_token` in memory or a secure cookie — avoid `localStorage`.
3. Attach `Authorization: Bearer <access_token>` to every protected request.
4. On a `401` response, use the `refresh_token` to obtain a new `access_token`.

### Video upload flow

1. Upload the source video to Cloudinary client-side to get a URL.
2. `POST /api/videos/upload?input_url=<url>&caption=<text>&tags=<comma-separated>` — receive a `video_id`.
3. Poll `GET /api/videos/:video_id/status` every few seconds.
4. When `status` is `"complete"`, `video_url` and `thumbnail_url` are ready to display.

### Error handling

Implement generic handlers for the following HTTP status codes at the network layer:

| Status | Meaning |
|--------|---------|
| `400` | Bad Request — check request body/params |
| `401` | Unauthorized — token missing or expired |
| `403` | Forbidden — insufficient permissions |
| `404` | Not Found |
| `500` | Server Error |

### Headers for every authenticated request

```
Content-Type: application/json
Authorization: Bearer <access_token>
```

---

*Document prepared for the UniBuzz Frontend Team — March 2026*
