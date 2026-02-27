Feature: A user can login with an RSA key < 1024bits
  Background:
    Given there exists an account with username "[user:user]" and password "password2" with an RSA key of 1023 bits
    And there exists an account with username "[user:to]" and password "password"
    And there exists an account with username "[user:cc]" and password "password"
    Then it succeeds
    And bridge starts
    Then it succeeds
    And the user logs in with username "[user:user]" and password "password2"
    And user "[user:user]" finishes syncing
    And user "[user:user]" connects and authenticates IMAP client "1"
    And user "[user:user]" connects and authenticates SMTP client "1"
    Then it succeeds

  Scenario: Login to account
    When the user logs in with username "[user:user]" and password "password2"
    Then user "[user:user]" is eventually listed and connected

  Scenario: Creates message to user's primary address
    When IMAP client "1" appends the following messages to "INBOX":
      | from               | to                   | subject | body |
      | john.doe@email.com | [user:user]@[domain] | foo     | bar  |
    Then it succeeds
    And IMAP client "1" eventually sees the following messages in "INBOX":
      | from               | to                   | subject | body |
      | john.doe@email.com | [user:user]@[domain] | foo     | bar  |
    And IMAP client "1" eventually sees the following messages in "All Mail":
      | from               | to                   | subject | body |
      | john.doe@email.com | [user:user]@[domain] | foo     | bar  |

  Scenario: Creates draft
    When IMAP client "1" appends the following messages to "Drafts":
      | from                 | to                 | subject | body |
      | [user:user]@[domain] | john.doe@email.com | foo     | bar  |
    Then it succeeds
    And IMAP client "1" eventually sees the following messages in "Drafts":
      | from                 | to                 | subject | body |
      | [user:user]@[domain] | john.doe@email.com | foo     | bar  |
    And IMAP client "1" eventually sees the following messages in "All Mail":
      | from                 | to                 | subject | body |
      | [user:user]@[domain] | john.doe@email.com | foo     | bar  |
  Scenario: Only from and to headers to internal account
    When SMTP client "1" sends the following message from "[user:user]@[domain]" to "[user:to]@[domain]":
      """
      From: Bridge Test <[user:user]@[domain]>
      To: Internal Bridge <[user:to]@[domain]>

      hello

      """
    Then it succeeds
    When user "[user:user]" connects and authenticates IMAP client "1"
    Then IMAP client "1" eventually sees the following messages in "Sent":
      | from                 | to                 | subject |
      | [user:user]@[domain] | [user:to]@[domain] |         |
    And the body in the "POST" request to "/mail/v4/messages" is:
      """
      {
        "Message": {
          "Subject": "",
          "Sender": {
            "Name": "Bridge Test"
          },
          "ToList": [
            {
              "Address": "[user:to]@[domain]",
              "Name": "Internal Bridge"
            }
          ],
          "CCList": [],
          "BCCList": [],
          "MIMEType": "text/plain"
        }
      }
      """
  Scenario: Basic message to internal account
    When SMTP client "1" sends the following message from "[user:user]@[domain]" to "[user:to]@[domain]":
      """
      From: Bridge Test <[user:user]@[domain]>
      To: Internal Bridge <[user:to]@[domain]>
      Subject: Plain text internal
      Content-Disposition: inline
      Content-Type: text/plain; charset=utf-8

      This is body of mail 👋

      """
    Then it succeeds
    When user "[user:user]" connects and authenticates IMAP client "1"
    Then IMAP client "1" eventually sees the following messages in "Sent":
      | from                 | to                 | subject             |
      | [user:user]@[domain] | [user:to]@[domain] | Plain text internal |
    And the body in the "POST" request to "/mail/v4/messages" is:
      """
      {
        "Message": {
          "Subject": "Plain text internal",
          "Sender": {
            "Name": "Bridge Test"
          },
          "ToList": [
            {
              "Address": "[user:to]@[domain]",
              "Name": "Internal Bridge"
            }
          ],
          "CCList": [],
          "BCCList": [],
          "MIMEType": "text/plain"
        }
      }
      """
