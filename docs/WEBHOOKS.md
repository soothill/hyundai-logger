# Webhook Notifications

Hyundai Logger supports sending notifications to various platforms via webhooks. This allows you to receive real-time alerts about your vehicle's status, charging events, and system errors.

## Supported Platforms

- **Slack** - Team communication
- **Discord** - Community and gaming chat
- **Generic Webhooks** - Any HTTP endpoint supporting JSON payloads

## Configuration

Add webhook configuration to your `config.yaml`:

```yaml
webhooks:
  enabled: true

  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
    channel: "#vehicle-alerts"        # Optional
    username: "Hyundai Logger"        # Optional
    icon_emoji: ":car:"               # Optional

  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/YOUR/WEBHOOK/URL"
    username: "Hyundai Logger"        # Optional
    avatar_url: "https://example.com/avatar.png"  # Optional

  generic:
    - name: "Custom Alert System"
      url: "https://your-server.com/webhook"
      method: "POST"                  # Optional, defaults to POST
      headers:                        # Optional custom headers
        Authorization: "Bearer YOUR_TOKEN"
        X-Custom-Header: "value"
```

## Setting Up Webhooks

### Slack

1. Go to your Slack workspace settings
2. Navigate to **Apps** → **Incoming Webhooks**
3. Click **Add New Webhook to Workspace**
4. Select the channel to post to
5. Copy the webhook URL and add it to your config

### Discord

1. Open Discord and go to your server
2. Navigate to **Server Settings** → **Integrations** → **Webhooks**
3. Click **New Webhook** or **Create Webhook**
4. Set the name and channel
5. Copy the webhook URL and add it to your config

### Generic Webhooks

Any HTTP endpoint that accepts POST requests with JSON payloads can be used. The payload format is:

```json
{
  "level": "info|warning|error|critical",
  "title": "Alert Title",
  "message": "Detailed message",
  "timestamp": "2025-11-01T10:30:00Z",
  "vehicle_vin": "5NPE24AF1KH123456",
  "metadata": {
    "key": "value"
  }
}
```

## Alert Levels

Hyundai Logger uses four alert levels:

- **Info** (🟢) - Normal operations, charging started, etc.
- **Warning** (🟡) - Minor issues, rate limiting, etc.
- **Error** (🔴) - API failures, authentication errors
- **Critical** (🔴) - System failures, consecutive errors threshold reached

## Example Notifications

### Slack
![Slack Example](../assets/slack-example.png)

Message format:
```
**Charging Started**
Your IONIQ 5 has started charging

Level: info
Time: 2025-11-01 10:30:00
Vehicle: 5NPE24AF1KH123456
```

### Discord
![Discord Example](../assets/discord-example.png)

Embedded rich message with color coding based on alert level.

## Use Cases

### 1. Charging Notifications
Get notified when your EV starts or stops charging:
```yaml
webhooks:
  enabled: true
  slack:
    enabled: true
    webhook_url: "YOUR_URL"
    channel: "#ev-charging"
```

### 2. Error Monitoring
Monitor API errors and system issues:
```yaml
webhooks:
  enabled: true
  discord:
    enabled: true
    webhook_url: "YOUR_URL"
```

### 3. Integration with Home Automation
Send alerts to your home automation system:
```yaml
webhooks:
  enabled: true
  generic:
    - name: "Home Assistant"
      url: "http://homeassistant.local:8123/api/webhook/hyundai_logger"
      headers:
        Authorization: "Bearer YOUR_HA_TOKEN"
```

### 4. PagerDuty Integration
Create custom integration for on-call alerts:
```yaml
webhooks:
  enabled: true
  generic:
    - name: "PagerDuty"
      url: "https://events.pagerduty.com/v2/enqueue"
      headers:
        Content-Type: "application/json"
        Authorization: "Token token=YOUR_INTEGRATION_KEY"
```

## Testing Webhooks

Use the configuration validation tool to test your webhooks:

```bash
./hyundai-logger-validate --config config.yaml --verbose
```

This will send test notifications to all configured webhooks and verify they're working.

## Troubleshooting

### Webhook Not Receiving Messages

1. **Check webhook URL** - Ensure the URL is correct and accessible
2. **Verify webhook is enabled** - Both global `webhooks.enabled` and individual webhook `enabled` must be true
3. **Check network connectivity** - Ensure your server can reach the webhook endpoint
4. **Review logs** - Check application logs for webhook errors

### Timeout Issues

Webhook requests have a 10-second timeout by default. If your endpoint is slow:

```yaml
webhooks:
  enabled: true
  timeout: 30s  # Increase timeout (not yet implemented)
```

### Rate Limiting

Some platforms (like Slack) have rate limits. If sending too many notifications:

- Adjust your polling intervals
- Use alert thresholds to reduce notification frequency
- Consider batching notifications (future feature)

## Security Best Practices

1. **Use HTTPS** - Always use secure webhook URLs
2. **Rotate URLs** - Periodically regenerate webhook URLs
3. **Limit Permissions** - Use read-only channels when possible
4. **Monitor Usage** - Review webhook logs for unauthorized use
5. **Use Environment Variables** - Don't commit webhook URLs to version control

Example using environment variables:

```yaml
webhooks:
  enabled: true
  slack:
    enabled: true
    webhook_url: "${SLACK_WEBHOOK_URL}"
```

Then set the environment variable:
```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

## Advanced Configuration

### Multiple Slack Channels

Currently, you can only configure one Slack webhook. To send to multiple channels, use multiple generic webhooks or configure Slack workflow automation.

### Custom Notification Filters

Future feature: Filter notifications by vehicle VIN, alert level, or event type.

### Retry Logic

Webhook requests are not automatically retried. If a webhook fails, the error is logged but the application continues.

## API Reference

### Alert Object

```go
type Alert struct {
    Level       AlertLevel              // info, warning, error, critical
    Title       string                  // Short title
    Message     string                  // Detailed message
    Timestamp   time.Time               // When the alert occurred
    VehicleVIN  string                  // Associated vehicle (optional)
    Metadata    map[string]interface{}  // Additional context (optional)
}
```

### Configuration Options

```go
type WebhooksConfig struct {
    Enabled bool                     // Master switch for all webhooks
    Slack   SlackWebhookConfig       // Slack configuration
    Discord DiscordWebhookConfig     // Discord configuration
    Generic []GenericWebhookConfig   // Array of generic webhooks
}
```

## Examples

See `config.example.yaml` for a complete configuration example with webhooks.

## Support

For issues or questions:
- GitHub Issues: https://github.com/soothill/hyundai-logger/issues
- Documentation: https://github.com/soothill/hyundai-logger/docs

## Future Enhancements

Planned features:
- Microsoft Teams webhooks
- Telegram bot integration
- Email notifications via SMTP (already supported via alerts)
- SMS via Twilio
- Webhook retry logic with exponential backoff
- Notification templates
- Alert filtering and routing rules
