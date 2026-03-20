# UniBuzz Video Post Feature — Frontend Integration Guide

This guide is written for the Flutter frontend team. It covers everything you need to know to integrate the video upload and feed features smoothly, including how to keep videos loading fast for users at a small-scale level.

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [The Upload Flow — How It Works](#the-upload-flow)
4. [Step-by-Step Integration](#step-by-step-integration)
   - [Step 1: Pick and Upload to Cloudinary Directly](#step-1-pick-and-upload-to-cloudinary-directly)
   - [Step 2: Submit the Video to the Backend](#step-2-submit-the-video-to-the-backend)
   - [Step 3: Poll for Processing Status](#step-3-poll-for-processing-status)
5. [Feed Integration](#feed-integration)
6. [Making Videos Load Fast (TikTok-Style)](#making-videos-load-fast)
7. [User Experience Guidelines](#user-experience-guidelines)
8. [Error Handling Reference](#error-handling-reference)
9. [Data Structures](#data-structures)

---

## Overview

UniBuzz uses an **asynchronous, URL-based video upload model**. This means:

- The app does **not** send the video file directly to the UniBuzz backend.
- Instead, the video is uploaded **directly from the device to Cloudinary** (a media cloud service), and then the resulting Cloudinary URL is sent to the backend.
- The backend queues a background job that processes the video (generates a thumbnail, finalizes storage) and marks it as ready.
- The feed only shows videos that have been fully processed.

This approach offloads heavy lifting away from the backend and keeps the API fast.

---

## Authentication

Every endpoint requires a valid JWT token in the request header.

**Header format:**

```
Authorization: Bearer <your_jwt_token>
```

There are no public endpoints in the video feature. Always attach the token. If the token is missing or expired, the server returns `401 Unauthorized`.

---

## The Upload Flow

Here is the full lifecycle of a video post from the user's perspective:

```
User picks video
      ↓
App uploads video directly to Cloudinary (client-side)
      ↓
App receives a Cloudinary video URL
      ↓
App calls POST /api/videos/upload with the URL, caption, and tags
      ↓
Backend acknowledges (202 Accepted) and returns a video_id
      ↓
App polls GET /api/videos/:id/status every few seconds
      ↓
When status = "processed", the video is live on the feed
```

The user does not need to wait for the processing to finish to continue using the app. You should let them know the video is being processed and move on.

---

## Step-by-Step Integration

### Step 1: Pick and Upload to Cloudinary Directly

The app must upload the video to Cloudinary **before** calling the backend. Cloudinary provides an unsigned upload API that the mobile client can use without exposing secrets.

**What you need from the backend team / DevOps:**
- The **Cloudinary Cloud Name**: `df3lhzzy7`
- An **unsigned upload preset** (ask the backend team to create one in the Cloudinary dashboard — this is required for client-side uploads and keeps credentials off the device)

**Cloudinary direct upload endpoint:**

```
POST https://api.cloudinary.com/v1_1/{cloud_name}/video/upload
```

**Form fields to send (multipart/form-data):**

| Field | Value |
|---|---|
| `file` | The video file bytes |
| `upload_preset` | The unsigned preset name (provided by backend team) |
| `resource_type` | `video` |
| `folder` | `unibuzz/videos` |

**What Cloudinary returns:**
- A `secure_url` field — this is the URL you pass to the backend in Step 2.
- A `public_id` field — you don't need this for the upload, but good to log.

**Upload progress:** Cloudinary's upload API supports progress tracking via the standard HTTP upload progress, which Flutter's `http` or `dio` packages both support. Use this to show the user a real upload progress bar.

**File size guidance:** For a small-scale app, keep videos under **50MB**. You should validate file size on the client before starting the upload and show a friendly message if it's too large. You can also compress the video on the device before uploading (see the [UX section](#user-experience-guidelines) for tips).

---

### Step 2: Submit the Video to the Backend

Once Cloudinary returns the `secure_url`, call the backend upload endpoint.

**Endpoint:**
```
POST /api/videos/upload
```

**All parameters go in the query string (not the request body):**

| Parameter | Required | Description |
|---|---|---|
| `input_url` | Yes | The URL-encoded Cloudinary video URL from Step 1 |
| `caption` | No | A text description of the video |
| `tags` | No | Comma-separated hashtags, e.g. `campus,football,fyp` |

**Tag rules:**
- Maximum of **10 tags**
- Tags are **case-insensitive** — the backend normalizes them to lowercase
- Duplicates are automatically removed
- Do not include the `#` symbol — send `campus` not `#campus`

**Important — URL encoding:** The `input_url` value must be URL-encoded because it contains slashes and special characters. Use your HTTP client's built-in query parameter encoding. Do not manually append the URL as a raw string.

**Expected response — 202 Accepted:**

```json
{
  "message": "video accepted and queued for processing",
  "video_id": "a1b2c3d4-...",
  "status": "pending",
  "tags": ["campus", "football"]
}
```

Save the `video_id` — you need it for Step 3.

**Error responses at this step:**

| Status | Reason |
|---|---|
| 400 | `input_url` is missing |
| 400 | More than 10 tags provided |
| 400 | URL is not a valid URL |
| 401 | Token is missing or expired |
| 500 | Backend database error |

---

### Step 3: Poll for Processing Status

After submitting, the video is being processed in the background. You need to poll the status endpoint to know when it's ready.

**Endpoint:**
```
GET /api/videos/:id/status
```

Replace `:id` with the `video_id` from Step 2.

**Response while processing:**

```json
{
  "video_id": "a1b2c3d4-...",
  "status": "pending",
  "video_url": null,
  "thumbnail_url": null
}
```

**Response when ready:**

```json
{
  "video_id": "a1b2c3d4-...",
  "status": "processed",
  "video_url": "https://res.cloudinary.com/...",
  "thumbnail_url": "https://res.cloudinary.com/..."
}
```

**Polling strategy:**
- Poll every **5 seconds** while the user is on a "processing" screen or notification
- Stop polling once `status` is `"processed"` or if the user navigates away
- Do not poll indefinitely — set a **maximum of 20 attempts** (about 100 seconds). If still pending after that, tell the user to check back later.
- Processing typically takes **15–45 seconds** depending on video length.

**Do not block the user's session waiting for this.** Let them continue using the app. You can use a background timer, a local notification, or a badge on their profile once it's done.

---

## Feed Integration

**Endpoint:**
```
GET /api/feed
```

**No query parameters are needed.** The feed always returns the **20 most recent processed videos**, newest first.

**Response — array of video objects:**

Each item in the array contains:

| Field | Type | Description |
|---|---|---|
| `id` | string (UUID) | Unique video ID |
| `user_id` | string (UUID) | ID of the poster |
| `username` | string | Username of the poster |
| `university_name` | string or null | University of the poster |
| `year_of_study` | integer or null | Year of study of the poster |
| `caption` | string | Video caption |
| `hashtags` | array of strings | Tags on the video |
| `video_url` | string | Direct video URL (Cloudinary CDN) |
| `thumbnail_url` | string | Thumbnail image URL |
| `created_at` | string (ISO 8601) | When the video was posted |

**Caching:** The backend caches the feed for **30 seconds**. This means back-to-back rapid calls return the same data. Do not spam the endpoint.

**Pagination:** Currently there is no pagination — it's always the latest 20. Design your feed UI to refresh on pull-to-refresh, not infinite scroll (until pagination is added).

---

## Making Videos Load Fast

This is the most important section for a smooth user experience. Since this is a small-scale app, you can achieve TikTok-like performance without a CDN of your own by using Cloudinary's built-in capabilities and smart Flutter-side strategies.

### 1. Use Cloudinary's Transformation URLs for Adaptive Streaming

All video URLs returned by the backend are standard Cloudinary CDN URLs. Cloudinary supports real-time transformations via URL parameters. You can modify the `video_url` before passing it to your video player to request an optimized version.

**Key Cloudinary URL transformations to apply:**

- **Adaptive quality:** Add `/q_auto` to automatically serve the best quality for the network. Example: insert `q_auto/` before the filename segment in the URL.
- **Format optimization:** Add `/f_auto` to serve the best format the device supports (e.g., WebM, MP4).
- **Resolution cap:** If you want to limit to 720p for data savings, add `/h_720,c_limit/`.

**How to apply:** The Cloudinary URL follows this pattern:
```
https://res.cloudinary.com/{cloud_name}/video/upload/{transformations}/{public_id}.mp4
```

You insert the transformations after `/upload/`. For a video with no transformations:
```
https://res.cloudinary.com/df3lhzzy7/video/upload/unibuzz/videos/abc123.mp4
```

With quality + format optimization:
```
https://res.cloudinary.com/df3lhzzy7/video/upload/q_auto,f_auto/unibuzz/videos/abc123.mp4
```

Apply this URL transformation on the Flutter side, not by modifying the backend — just manipulate the string before passing it to your video player. This is a simple string insertion.

---

### 2. Preload the Next Video While the Current One Plays

The single biggest performance improvement you can make is **preloading**. When the user is watching video N, silently initialize the player for video N+1 in the background.

**Strategy:**
- Keep a list of the 20 feed videos in memory.
- Track the current index.
- When the user is 70% through the current video, start buffering the next one.
- Use a pool of 2–3 video controllers — the current, the next, and possibly the previous. Dispose of controllers that are more than 2 positions away.

This is exactly how TikTok works. The video feels instant because it's already loaded before the user swipes.

---

### 3. Show Thumbnails Immediately

Every video has a `thumbnail_url` in the feed response. Use it as a placeholder while the video loads.

**The pattern:**
- When the feed loads, immediately display all thumbnails (images load much faster than video).
- When the video for the current item is buffered and ready to play, fade the thumbnail out.
- This gives the illusion of zero load time.

Cache the thumbnails aggressively using a Flutter image caching package. Thumbnails are small JPEG files and should stay in memory for the entire session.

---

### 4. Cache the Feed Response Locally

The backend caches the feed for 30 seconds. On the Flutter side, you should cache it longer for offline or slow-network scenarios.

**Recommended approach:**
- Store the last successful feed response in local storage (e.g., using `shared_preferences` or `hive`).
- On app launch or feed open, immediately render the cached feed while fetching a fresh one in the background.
- Swap the content when the new data arrives (pull-to-refresh pattern).

This makes the feed appear to load instantly even on cold start.

---

### 5. Use a PageView for the Feed

Model your feed as a vertical `PageView` (one video per page), not a `ListView`. This is the TikTok pattern. Each page fills the screen, snaps into place, and you manage the lifecycle (play/pause) based on the current page index.

**Controller lifecycle:**
- Only the **current page** should be playing.
- The page before and after should be initialized and buffered but paused.
- Pages more than 1 away should be disposed of to save memory.

---

### 6. Keep Uploads Small — Compress Before Upload

Video file size directly impacts how fast the upload step feels. On the Flutter side, before sending to Cloudinary:

- Use a video compression library to reduce file size while keeping acceptable quality.
- Target a bitrate suitable for mobile — around 2–4 Mbps is sufficient for a social app at 720p.
- Trim the video to a maximum length on the client (e.g., 60 seconds) and show a clear error if the user tries to upload longer.
- Give users visual feedback (a progress bar tied to Cloudinary's upload progress) so the upload doesn't feel frozen.

---

### 7. Pause Videos When Not Visible

Always pause the video player when:
- The app goes to the background (`AppLifecycleState.paused`)
- The user navigates away from the feed screen
- The video is no longer the active page in the PageView

This avoids wasted bandwidth and battery, and prevents audio bleed from multiple videos.

---

## User Experience Guidelines

### Upload Screen
- Show a **progress bar** during the Cloudinary upload phase (this is the longest step).
- After the backend accepts the upload (202 response), show a **"Your video is being processed"** message and let the user leave.
- Do not make the user wait on a loading screen for processing to finish.
- If upload fails (network error or Cloudinary error), allow the user to **retry** without re-picking the video.

### Processing Feedback
- Consider showing a **"Processing..."** badge on the user's profile or a post-upload confirmation screen.
- If the user checks their own profile and the video is still pending, show a spinner/skeleton instead of a broken thumbnail.

### Feed Screen
- Show a full-screen skeleton loader (matching the video card layout) on first load.
- Show thumbnails immediately while videos buffer.
- Add a mute/unmute button — many users scroll in public and expect videos to be muted by default.
- Loop short videos automatically.

### Tags/Caption Screen
- Show a tag input that strips the `#` character automatically — the backend does not want it.
- Enforce the 10-tag limit on the client before sending and show a friendly counter (e.g., "3/10 tags").
- Caption is optional — do not block upload if it's empty.

---

## Error Handling Reference

| HTTP Status | What it means | What to show the user |
|---|---|---|
| 400 | Bad request (missing URL, too many tags, invalid URL) | Show the specific validation message |
| 401 | Not authenticated / token expired | Redirect to login screen |
| 404 | Video ID not found (status poll) | Show "Something went wrong" and stop polling |
| 500 | Server error | Show "Server error, please try again later" |
| Network error | No connectivity | Show offline banner, retry with exponential backoff |

For the upload step specifically, if Cloudinary itself returns an error (before you even call the backend), show the user a retry option — do not advance to Step 2.

---

## Data Structures

### Feed Video Item

```
id              → String (UUID)
user_id         → String (UUID)
username        → String
university_name → String? (nullable)
year_of_study   → int? (nullable)
caption         → String
hashtags        → List<String>
video_url       → String (Cloudinary CDN URL)
thumbnail_url   → String (Cloudinary CDN URL)
created_at      → DateTime (ISO 8601)
```

### Video Status Response

```
video_id        → String (UUID)
status          → String ("pending" | "processed")
video_url       → String? (null while pending)
thumbnail_url   → String? (null while pending)
```

### Upload Response (202)

```
message         → String
video_id        → String (UUID) ← save this for status polling
status          → String ("pending")
tags            → List<String>
```

---

## Quick Reference — Endpoints

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/videos/upload` | Submit video URL to backend |
| GET | `/api/videos/:id/status` | Poll processing status |
| GET | `/api/feed` | Get the 20 latest videos |

All requests must include `Authorization: Bearer <token>`.

---

*This guide reflects the current state of the backend as of March 2026. If the backend team introduces cursor-based pagination or chunked upload support, this document will be updated.*
