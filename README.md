# Walzen NTFY Toaster for Windows

Unfortunately, ntfy has no native super simple Windows client to receive notifications.
This small app aims to solve this!

You can subscribe to topics via a config.yaml file.

Feature parity with the PWA client is almost there.

We support
- Priority
- Notify about presence of file attachments
- Display button to go to configured click URL
- Emoji tags

Notifations are retrieved via JSON stream. A reconnection attempt will be made every 15 seconds if a server is unreachable.

## Example

Comparison with PWA notification:

![WebNormal](https://github.com/Walzen-Group/ntfy-toaster/assets/18438899/502e4e44-6fa6-4b5a-b933-dcfecca37153)
![ClientNormal](https://github.com/Walzen-Group/ntfy-toaster/assets/18438899/2ea0291e-0345-4f65-8065-0228e4c89bd9)

Additional features:

![clickurl](https://github.com/Walzen-Group/ntfy-toaster/assets/18438899/6dce23a1-5f77-438b-add6-9e4a4d77c80b)
![attachment](https://github.com/Walzen-Group/ntfy-toaster/assets/18438899/73e9a595-6a50-4b87-9045-2dd972d964a0)


## Installation

Download executable and run. In the tray, you can right click on the ntfy icon and select "Open Config", which will open the app directory for you.

## Configuration

The `config.yaml` configuration file is located in the app directory. You can subscribe to topics by adding them to the `topics` list.
If the app is already running, there is no need to restart it, as the config file will be reloaed on save.

```yaml
topics:
  thesuperdupertopic:
    token: tk_mytoken1
    url: https://ntfy.sh/topic1
  thetopicest:
    token: ""
    url: https://ntfy.sh/topic2
```

## Command line output

~~If you want to see the raw output or error / info messages, you can run the app in a terminal.~~

Currently not working.

## Attribution

- ntfy.sh - The one and only
- golang toast (https://github.com/go-toast/toast, https://github.com/ShinyTrash/abc) - Interface to deliver toast notifications in golang
- systray - lightweight golang system tray library
- yaml - library to parse yaml in golang
