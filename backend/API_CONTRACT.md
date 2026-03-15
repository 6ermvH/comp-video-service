# Backend API Contract (Agent 2)

Base path: `/api`

## Public respondent flow

### `POST /session/start`
Request:
```json
{
  "study_id": "uuid",
  "device_type": "desktop",
  "browser": "chrome",
  "role": "general_viewer",
  "experience": "limited"
}
```
Response:
```json
{
  "session_token": "string",
  "assigned": 20,
  "meta": {
    "study_id": "uuid",
    "study_name": "Flooding Benchmark",
    "effect_type": "flooding",
    "max_tasks_per_participant": 20,
    "tie_option_enabled": true,
    "reasons_enabled": true,
    "confidence_enabled": true
  },
  "first_task": {
    "presentation_id": "uuid",
    "source_item_id": "uuid",
    "task_order": 1,
    "is_attention_check": false,
    "is_practice": false,
    "left": {"id": "uuid", "presigned_url": "..."},
    "right": {"id": "uuid", "presigned_url": "..."}
  }
}
```

### `GET /session/:token/next-task`
- `200` + task payload
- `204` when no tasks remain

### `POST /task/:id/response`
Request:
```json
{
  "choice": "left",
  "reason_codes": ["realism", "physics"],
  "confidence": 4,
  "response_time_ms": 5400,
  "replay_count": 1
}
```

### `POST /task/:id/event`
Request:
```json
{
  "event_type": "replay_clicked",
  "payload_json": {"key": "value"}
}
```

### `POST /session/:token/complete`
Response:
```json
{"completion_code":"CVS-xxxxxxxx"}
```

## Admin auth

### `POST /admin/login`
Response:
```json
{
  "token": "jwt",
  "csrf_token": "hex-string",
  "admin": {"id":"uuid","username":"admin"}
}
```

For mutating admin requests (`POST/PUT/PATCH/DELETE`), send both:
- `Authorization: Bearer <token>`
- `X-CSRF-Token: <csrf_token>`

## Admin data endpoints

- `GET /admin/studies`
- `POST /admin/studies`
- `PATCH /admin/studies/:id`
- `POST /admin/studies/:id/groups`
- `POST /admin/studies/:id/import` (multipart `file` CSV)
- `POST /admin/assets/upload` (multipart `file`, `method_type`, optional `source_item_id`)
- `GET /admin/source-items?study_id=<uuid>&group_id=<uuid>`

## Admin analytics and export

- `GET /admin/analytics/overview`
- `GET /admin/analytics/study/:id`
- `GET /admin/analytics/qc`
- `GET /admin/export/csv`
- `GET /admin/export/json`

Export CSV/JSON rows use fixed 14 fields:
1. `response_id`
2. `participant_id`
3. `session_token`
4. `study_id`
5. `pair_presentation_id`
6. `source_item_id`
7. `task_order`
8. `choice`
9. `reason_codes`
10. `confidence`
11. `response_time_ms`
12. `replay_count`
13. `is_attention_check`
14. `created_at`
