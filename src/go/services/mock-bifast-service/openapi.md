```
openapi: 3.0.0
info:
  title: Mock BI-FAST Service
  version: v1
paths:
  /bifast/transfer:
    post:
      summary: Simulate external transfer request
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                request_id:
                  type: string
                amount:
                  type: integer
                from_account:
                  type: string
                to_account:
                  type: string
                currency:
                  type: string
      responses:
        '200':
          description: Received transfer request (mock external)
          content:
            application/json:
              schema:
                type: object
                properties:
                  external_ref_id:
                    type: string
                  status:
                    type: string
                    enum: [PENDING, SUCCESS, FAILED]
                  message:
                    type: string

  /bifast/callback:
    post:
      summary: Callback notification from external bank
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                external_ref_id:
                  type: string
                status:
                  type: string
                  enum: [SUCCESS, FAILED]
                reason:
                  type: string

  /bifast/status/{external_ref_id}:
    get:
      summary: Get status of mock external transfer
      parameters:
        - name: external_ref_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Current status for transfer
          content:
            application/json:
              schema:
                type: object
                properties:
                  external_ref_id:
                    type: string
                  status:
                    type: string
                    enum: [PENDING, SUCCESS, FAILED]
                  last_updated:
                    type: string
                    format: date-time

```