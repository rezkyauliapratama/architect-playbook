{
  "openapi": "3.0.3",
  "info": {
    "title": "Notification Service API",
    "version": "1.0.0",
    "description": "API for creating and retrieving notifications for users."
  },
  "paths": {
    "/notifications": {
      "post": {
        "summary": "Create a new notification",
        "operationId": "createNotification",
        "tags": ["Notifications"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/CreateNotificationRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Notification created successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/NotificationResponse"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request body",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "error": { "type": "string" }
                  }
                }
              }
            }
          },
          "500": {
            "description": "Internal server error",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "error": { "type": "string" }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/notifications/{recipientId}": {
      "get": {
        "summary": "Get notifications for a specific recipient",
        "operationId": "getNotifications",
        "tags": ["Notifications"],
        "parameters": [
          {
            "name": "recipientId",
            "in": "path",
            "required": true,
            "schema": { "type": "string" },
            "description": "Recipient identifier"
          },
          {
            "name": "channel",
            "in": "query",
            "schema": { "type": "string" },
            "description": "Filter by notification channel"
          },
          {
            "name": "app",
            "in": "query",
            "schema": { "type": "string" },
            "description": "Filter by application source"
          },
          {
            "name": "limit",
            "in": "query",
            "schema": { "type": "integer", "default": 10 },
            "description": "Limit number of results"
          },
          {
            "name": "offset",
            "in": "query",
            "schema": { "type": "integer", "default": 0 },
            "description": "Offset for pagination"
          }
        ],
        "responses": {
          "200": {
            "description": "List of notifications retrieved successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/GetNotificationsResponse"
                }
              }
            }
          },
          "500": {
            "description": "Internal server error",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "error": { "type": "string" }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "CreateNotificationRequest": {
        "type": "object",
        "required": ["recipientId", "type", "title", "message", "channel", "app"],
        "properties": {
          "recipientId": { "type": "string", "example": "user-123" },
          "type": { "type": "string", "enum": ["EMAIL", "SMS", "PUSH"], "example": "EMAIL" },
          "title": { "type": "string", "example": "Payment Successful" },
          "message": { "type": "string", "example": "Your payment of IDR 100,000 has been completed." },
          "channel": { "type": "string", "example": "transaction" },
          "app": { "type": "string", "example": "payment-service" },
          "data": {
            "type": "object",
            "additionalProperties": true,
            "example": { "transactionId": "TX12345" }
          }
        }
      },
      "NotificationResponse": {
        "type": "object",
        "properties": {
          "notificationId": { "type": "string", "example": "notif-abc123" },
          "recipientId": { "type": "string", "example": "user-123" },
          "type": { "type": "string", "example": "EMAIL" },
          "title": { "type": "string", "example": "Payment Successful" },
          "message": { "type": "string", "example": "Your payment of IDR 100,000 has been completed." },
          "channel": { "type": "string", "example": "transaction" },
          "app": { "type": "string", "example": "payment-service" },
          "status": { "type": "string", "example": "SENT" },
          "createdAt": { "type": "string", "format": "date-time", "example": "2025-10-18T08:00:00Z" },
          "sentAt": { "type": "string", "format": "date-time", "nullable": true },
          "data": {
            "type": "object",
            "additionalProperties": true,
            "example": { "transactionId": "TX12345" }
          }
        }
      },
      "GetNotificationsResponse": {
        "type": "object",
        "properties": {
          "notifications": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/NotificationResponse" }
          },
          "total": { "type": "integer", "example": 100 },
          "limit": { "type": "integer", "example": 10 },
          "offset": { "type": "integer", "example": 0 }
        }
      }
    }
  }
}
