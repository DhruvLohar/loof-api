# API Documentation

Base URL prefix: `/v1`

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

---

### POST `/v1/users/resend-otp`

**Request Payload**
```json
{
  "uid": "uint"
}
```

**Response Payload**
```json
{
  "success": true,
  "message": "OTP sent successfully",
  "otp": "int" // TODO: remove once whatsapp integration is complete
}
```

---

### POST `/v1/users/verify-otp`

**Request Payload**
```json
{
  "uid": "uint",
  "otp": "int"
}
```

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

---

### POST `/v1/users/validate-username`

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
