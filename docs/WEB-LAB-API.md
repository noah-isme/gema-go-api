# Web Lab API

The Web Lab endpoints allow students to download assignment briefs and upload HTML/CSS/JS projects for automated validation.

## List assignments

`GET /api/v2/web-lab/assignments`

```
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "message": "assignments retrieved",
  "data": [
    {
      "id": 3,
      "title": "Landing Page Fundamentals",
      "requirements": "Bangun halaman responsive dengan hero, fitur, dan CTA.",
      "assets": [
        "assets/style-guide.pdf",
        "assets/hero.png"
      ],
      "rubric": "Struktur HTML 40%, Styling 35%, Interaktivitas 25%",
      "created_at": "2025-01-08T07:41:16Z",
      "updated_at": "2025-01-08T07:41:16Z"
    }
  ]
}
```

## Retrieve assignment detail

`GET /api/v2/web-lab/assignments/{id}`

Returns a single assignment in the same schema as the list endpoint.

## Submit a project archive

`POST /api/v2/web-lab/submissions`

This endpoint is protected by JWT. The authenticated student's `user_id` claim is used as the submitter ID.

### Request

`multipart/form-data`

| Field           | Type   | Required | Description                                 |
|-----------------|--------|----------|---------------------------------------------|
| `assignment_id` | number | Yes      | ID of the assignment being submitted        |
| `file`          | file   | Yes      | `.zip` archive containing HTML/CSS/JS files |

### Example response

```
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "message": "submission processed",
  "data": {
    "id": 12,
    "assignment_id": 3,
    "student_id": 42,
    "zip_url": "https://cdn.example.com/submissions/landing-page-42.zip",
    "status": "validated",
    "feedback": "Automated lint + Lighthouse heuristics lolos tanpa temuan.\nPerkiraan skor Lighthouse: 95/100",
    "score": 95,
    "created_at": "2025-01-08T07:52:10Z",
    "updated_at": "2025-01-08T07:52:10Z"
  }
}
```

### ZIP structure guideline

```
landing-page.zip
├── index.html
├── styles/
│   └── style.css
└── scripts/
    └── app.js
```

* The archive must be smaller than 10 MB.
* Executable files (`.exe`) and symbolic links are rejected.
* At least one HTML file is required to avoid a failing score.
