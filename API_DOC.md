# API Documentation

Base URL prefix: `/v1`

---

## Authentication

User endpoints marked **Protected** require the JWT returned by `/v1/users/verify-otp`, sent as a bearer token:

```
Authorization: Bearer <access_token>
```

The scheme is matched case-insensitively. A missing header, a non-`Bearer` scheme, an empty token, or an invalid/expired token all return:

```json
{
  "success": false,
  "message": "unauthorized"
}
```

with status `401`. Tokens are signed with HS256 and expire 24 hours after issue; call `/v1/users/verify-otp` again to get a new one. Cookies/sessions are not used for user authentication.

---

## Users

### POST `/v1/users/signup-signin`

**Request Payload**
```json
{
  "country_code": "string",
  "phone_number": "string"
}
```

**Response Payload**
```json
{
  "success": true,
  "data": {
    "id": "uint",
    "country_code": "string",
    "phone_number": "string"
  }
}
```

`id` is the user identifier used by every other user endpoint (`resend-otp`, `verify-otp`).

---

### POST `/v1/users/resend-otp`

**Request Payload**
```json
{
  "id": "uint"
}
```

**Response Payload**
```json
{
  "success": true,
  "message": "OTP sent successfully"
}
```

---

### POST `/v1/users/verify-otp`

**Request Payload**
```json
{
  "id": "uint",
  "otp": "int"
}
```

> **Note:** OTP verification is currently stubbed — any request with `otp` set to `123456` is accepted. TODO: verify against the generated OTP once whatsapp integration is complete.

**Response Payload**
```json
{
  "success": true,
  "message": "OTP verified successfully",
  "data": {
    "id": "uint",
    "phone_number": "string",
    "country_code": "string",
    "access_token": "string"
  }
}
```

Store `access_token` and send it as `Authorization: Bearer <access_token>` on every protected endpoint.

---

### POST `/v1/users/validate-username`

**Auth:** Protected

**Request Payload**
```json
{
  "username": "string"
}
```

**Response Payload**
```json
{
  "success": true,
  "message": "username is valid"
}
```

---

### POST `/v1/users/preferences`

**Auth:** Protected

**Request Payload**
```json
{
  "send_notification": "bool"
}
```

**Response Payload**
```json
{
  "success": true,
  "message": "preferences updated successfully",
  "data": {
    "send_notification": "bool"
  }
}
```

---

### GET `/v1/users/profile`

**Auth:** Protected

**Request Payload**
None

**Response Payload**
```json
{
  "success": true,
  "user": {
    "id": "uint",
    "name": "string",
    "username": "string",
    "country_code": "string",
    "phone_number": "string",
    "gender": "string",
    "dob": "date",
    "country": "string",
    "profile_picture": "string",
    "interests": ["string"],
    "cover_images": ["string"],
    "preferences": {},
    "is_active": "bool",
    "created_at": "datetime",
    "updated_at": "datetime",
    "last_login_at": "datetime"
  }
}
```

---

### POST `/v1/users/profile`

**Auth:** Protected

**Request Payload**
`multipart/form-data`, all fields optional — only the ones sent are updated.

| Field | Type | Notes |
|---|---|---|
| `name` | text | |
| `username` | text | must be unique |
| `gender` | text | |
| `dob` | text | `YYYY-MM-DD` |
| `country` | text | |
| `interests` | text(s) | repeat the field once per interest (`interests=music&interests=travel`), or send a single JSON array (`["music","travel"]`). `interests[]` is accepted too. Replaces the existing list; send one empty value to clear it |
| `profile_picture` | file | single image, uploaded to S3 |
| `cover_images` | file(s) | one or more images; replaces the existing list, uploaded to S3 |

**Response Payload**
```json
{
  "success": true,
  "message": "profile updated successfully",
  "data": {
    "id": "uint",
    "name": "string",
    "username": "string",
    "country_code": "string",
    "phone_number": "string",
    "gender": "string",
    "dob": "date",
    "country": "string",
    "profile_picture": "string",
    "interests": ["string"],
    "cover_images": ["string"],
    "preferences": {},
    "is_active": "bool",
    "created_at": "datetime",
    "updated_at": "datetime",
    "last_login_at": "datetime"
  }
}
```
