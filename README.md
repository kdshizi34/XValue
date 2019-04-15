# DCC-GO

Official golang implementation of the XValue DCC.

XValue is an inclusive public blockchain that provides the infrastructure and architecture for fully fledged financial functions on the blockchain. To learn more about XValue, please visit official website at <https://www.xvalue.org> or read the [White Paper](https://github.com/XValueFoundation/Whitepaper/blob/master/XValue%20Whitepaper%20V.1.1.pdf) and [Yellow Paper](https://github.com/XValueFoundation/Whitepaper/blob/master/DCCP%20Yellow%20Paper.pdf).

*Note: go-xvalue is considered beta software. We make no warranties or guarantees of its security or stability.*

## Building

### Supported Operating Systems

XValue DCC client currently supports the following 64bits operating systems:  

1. Ubuntu 16.04 or higher(18.04 recommended)
2. Debian 9
3. Centos 7 and RHEL 7
4. Fedora 25 or higher
5. MacOS Darwin 10.12 or higher

The following building instructions are based on Ubuntu 18.04(64bits), for more prerequisites and detailed build instructions please read the [Installation Instructions](https://github.com/XValueFoundation/xvalue-go/wiki/) on the wiki.

### Requirements

- [Go](https://golang.org/doc/install) version 1.9 or higher

 Ensure Go with the supported version is installed properly:

```bash
go version
```

### Environment

Ensure `$GOPATH` set to your work directory:

```bash
go env GOROOT GOPATH
```

### Get the source code

``` bash
git clone https://github.com/XValueFoundation/xvalue-go.git $GOPATH/src/github.com/xvalue-go
```

### Build

Once the dependencies are installed, run:

``` bash
make xvc
```

When successfully building the project, the `xvc` binary should be present in `build/bin` directory.

## Running

Currently, xvalue is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `xvc`. This section won't cover all the commands, for more information, please get more from the help of  command, e.g., `xvc help`.

```help
USAGE:
   xvc [options] command [command options] [arguments...]

COMMANDS:
   account           Manage accounts
   attach            Start an interactive JavaScript environment (connect to node)
   bug               opens a window to report a bug on the xvc repo
   console           Start an interactive JavaScript environment
   copydb            Create a local chain from a target chaindata folder
   dump              Dump a specific block from storage
   dumpconfig        Show configuration values
   export            Export blockchain into file
   export-preimages  Export the preimage database into an RLP stream
   import            Import a blockchain file
   import-preimages  Import the preimage database from an RLP stream
   init              Bootstrap and initialize a new genesis block
   js                Execute the specified JavaScript files
   license           Display license information
   monitor           Monitor and visualize node metrics
   removedb          Remove blockchain and state databases
   version           Print version numbers
   help, h           Shows a list of commands or help for one command

```

This command will start goxvc in fast sync mode, causing it to download block data and connect to the XValue network:

```bash
xvc
```

*Note: Do not perform cross-chain transfer, swap, deposit, and lock-in operations on the main network to avoid losing your tokens, such as: btc, eth.*

### Configuration

As an alternative to passing the numerous flags to the `xvc` binary, you can also pass a configuration file via:

```bash
xvc --config /path/to/your_config.toml
```

### JSON-RPC  API

Go-XValue has built-in support for a JSON-RPC based APIs ([standard APIs](https://github.com/XValueFoundation/go-xvalue/wiki/JSON-RPC)). These can be exposed via HTTP, WebSockets and IPC.The IPC interface is enabled by default and exposes all the APIs supported by go-xvalue, the goxvc node doesn't start the http and weboscket service and not all functionality is provided over these interfaces due to security reasons. These can be turned on/off and configured with the --rpcapi and --wsapi arguments when the goxvc node is started.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface
  * `--rpcport` HTTP-RPC server listening port
  * `--rpcapi` API's offered over the HTTP-RPC interface
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface
  * `--wsport` WS-RPC server listening port
  * `--wsapi` API's offered over the WS-RPC interface
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface
  * `--ipcpath` Filename for IPC socket/pipe within the datadir

Developer need to use your own programming environments' capabilities (libraries, tools, etc) to connect via HTTP, WS or IPC to a go-xvalue node configured with the above flags and you'll need to speak [JSON-RPC](http://www.jsonrpc.org/specification) on all transports. You can reuse the same connection for multiple requests!

## Testing

Please read the [User-Test-Guide](https://github.com/XValueFoundation/xvalue-go/wiki/User-Test-Guide) on the wiki.

## Contribution

Welcome anyone contribute to XValue with [issues](https://github.com/XValueFoundation/xvalue-go/issues) and [PRs](https://github.com/XValueFoundation/xvalue-go/pulls). Filing issues for problems you encounter is a great way to contribute. Contributing bugfix is greatly appreciated.

We recommend the following workflow:

1. Create an issue for your work.
2. Create a personal fork of the repository on GitHub.
3. Commit your changes.
4. Create a pull request (PR) against the upstream repository's master branch.

 If you wish to submit more complex changes though, please check up with the core devs first to ensure those changes are in line with the general philosophy of the project and/or get some early feedback which can make both your efforts much lighter as well as our review and merge procedures quick and simple.

## License

The xvalue-go project is licensed under the [GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html).
