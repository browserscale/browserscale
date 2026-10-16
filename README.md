<div align="center">

<img src="https://browserscale.cloud/icon-512.png" width="72" height="72" alt="browserscale">

# browserscale studio

**Describe the work. Get a bot that runs it.**

A desktop app for the web portals and back-office apps that have no API. Studio's agent walks the site with you in a live cloud browser, writes the automation as code your team can review, and runs it across as many browsers as the work needs.

![Windows](https://img.shields.io/badge/Windows-0a0a0a?style=flat-square&logo=windows)
![macOS](https://img.shields.io/badge/macOS-0a0a0a?style=flat-square&logo=apple)
![Linux](https://img.shields.io/badge/Linux-0a0a0a?style=flat-square&logo=linux&logoColor=white)

[browserscale.cloud/studio](https://browserscale.cloud/studio) · [Guide](https://browserscale.cloud/docs/guides/studio) · [Discord](https://discord.gg/SfE9C9K28D)

</div>

---

## Install

Download the installer for your system from the [latest release](https://github.com/browserscale/browserscale/releases/latest):

| | |
| --- | --- |
| **Windows** | `browserscale-studio_<version>_windows_<arch>-setup.exe` |
| **macOS** | `browserscale-studio_<version>_darwin_<arch>.dmg` (`arm64` for Apple silicon) |
| **Linux** | `.deb`, or the `.AppImage` for other distributions |

Or with npm, on any platform with Node 18 or newer:

```sh
npm i -g browserscale
browserscale            # opens Studio
```

Building bots needs [Go](https://go.dev/dl/) on the same machine. On Linux, Studio needs GTK 4 and WebKitGTK 6.0, as on Ubuntu 24.04 or Debian 13 and newer.

## How it works

| | |
| --- | --- |
| **Brief** | You and the agent settle what the bot does on which site, what it needs from you, and which outcomes count as success. |
| **Explore** | The agent goes through the site in a live browser and maps its screens and flows as it goes. You see each step and can take over. |
| **Build** | The agent writes the bot as a plain Go module, runs it, reads how each job ended, and fixes what broke. |
| **Run** | Runs start as many jobs as the work has, each in its own browser, and the agent reads the results with you. |

The bot builds into one program that runs without Studio and without a model. Bring your own model: Anthropic, OpenAI, Gemini or any OpenAI-compatible server, each with your own key.

## On the command line

The `browserscale` command covers what Studio does, for terminals and scripts. It shares Studio's sign-in, and a run started in one shows up live in the other.

```text
browserscale open [project]           Studio, at the project
browserscale run <project>            build, run, follow the log
browserscale runs <project>           a project's runs; show, stop
browserscale browsers                 running browsers; stop, live
browserscale browservm run <session>  a script beside the page
```

`browserscale help` lists everything. Without a credential store, as on a Linux server, it reads the key from `BROWSERSCALE_API_KEY`.

## Questions and bugs

Ask on [Discord](https://discord.gg/SfE9C9K28D) or open an [issue](https://github.com/browserscale/browserscale/issues).

## License

Studio and the `browserscale` command are free to use, under the terms in [LICENSE](https://github.com/browserscale/browserscale/blob/main/LICENSE). They are not open source. The bots you build with them are yours.
