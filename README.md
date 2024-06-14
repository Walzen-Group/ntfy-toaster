# Walzen NTFY Toaster for Windows

Unfortunately, ntfy has no native windows client to receive notifications.
This small app solves this!

You can subscribe to topics via a config.yaml file.

## Installation

Download executable and run. In the tray, you can right click on the ntfy icon and select "Open Config", which will open the app directory for you.

## Configuration

The `config.yaml`configuration file is located in the app directory. You can subscribe to topics by adding them to the `topics` list.
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

If you want to see the raw output or error / info messages, you can run the app in a terminal.
