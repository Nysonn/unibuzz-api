# Unibuzz API — Comments Endpoints

**Base URL prefix:** `/api`
**Auth:** All endpoints require an `Authorization: Bearer <token>` header.

---

## 1. Create a Comment

**`POST /api/videos/:id/comments`**

**URL Param:** `:id` — UUID of the video

**Request Body:**
```json
{
  "comment": "This is my comment"
}
```
> `comment` is required, max 500 characters.

**Responses:**

| Status | Body |
|--------|------|
| `201 Created` | `{ "message": "comment added", "comment_id": "<uuid>" }` |
| `400 Bad Request` | `{ "error": "invalid request" }` |
| `403 Forbidden` | `{ "error": "comments are disabled for this video" }` |
| `404 Not Found` | `{ "error": "video not found" }` |

---

## 2. Get Comments for a Video

**`GET /api/videos/:id/comments`**

**URL Param:** `:id` — UUID of the video

**No request body.**

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` (comments enabled) | `{ "comments_disabled": false, "comments": [ { "id": "<uuid>", "user_id": "<uuid>", "video_id": "<uuid>", "content": "...", "created_at": "...", "updated_at": "..." } ] }` |
| `200 OK` (comments disabled) | `{ "comments_disabled": true, "comments": [] }` |
| `404 Not Found` | `{ "error": "video not found" }` |

> Returns up to **20 most recent** comments. Comments matching the video owner's keyword filters are automatically hidden.

---

## 3. Toggle Comments On/Off

**`PATCH /api/videos/:id/comments/toggle`**

**URL Param:** `:id` — UUID of the video

**No request body.** Each call flips the current state.

> Only the **video owner** can call this endpoint.

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` | `{ "message": "comments enabled" \| "comments disabled", "comments_disabled": true \| false }` |
| `403 Forbidden` | `{ "error": "only the video owner can toggle comments" }` |
| `404 Not Found` | `{ "error": "video not found" }` |

---

## 4. Update a Comment

**`PUT /api/comments/:comment_id`**

**URL Param:** `:comment_id` — UUID of the comment

**Request Body:**
```json
{
  "comment": "Updated comment text"
}
```
> `comment` is required, max 500 characters. Only the **comment author** can update it.

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` | `{ "message": "comment updated" }` |
| `400 Bad Request` | `{ "error": "invalid request" }` |
| `403 Forbidden` | `{ "error": "not allowed or comment not found" }` |

---

## 5. Delete a Comment

**`DELETE /api/comments/:comment_id`**

**URL Param:** `:comment_id` — UUID of the comment

**No request body.** Only the **comment author** can delete it (soft delete).

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` | `{ "message": "comment deleted" }` |
| `403 Forbidden` | `{ "error": "not allowed or comment not found" }` |

---

## 6. Comment Keyword Filters (Video Owner Settings)

These allow a video owner to hide comments containing specific words across all their videos.

### Get All Filters

**`GET /api/me/comment-filters`**

**No request body.**

**Response `200 OK`:**
```json
[
  { "id": "<uuid>", "keyword": "spam", "created_at": "..." }
]
```

---

### Add a Keyword Filter

**`POST /api/me/comment-filters`**

**Request Body:**
```json
{
  "keyword": "spam"
}
```
> `keyword` is required, 1–100 characters. Max **50 filters** per user. Keywords are stored case-insensitively.

**Responses:**

| Status | Body |
|--------|------|
| `201 Created` | `{ "message": "filter added", "filter_id": "<uuid>" }` |
| `400 Bad Request` | `{ "error": "maximum of 50 filter keywords reached" }` |
| `409 Conflict` | `{ "error": "keyword already exists in your filter list" }` |

---

### Remove a Keyword Filter

**`DELETE /api/me/comment-filters/:filter_id`**

**URL Param:** `:filter_id` — UUID of the filter

**No request body.**

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` | `{ "message": "filter removed" }` |
| `404 Not Found` | `{ "error": "filter not found" }` |
