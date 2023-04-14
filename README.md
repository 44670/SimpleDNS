# SimpleDNS

SimpleDNS is a lightweight and easy-to-use DNS server written in Go. It listens for incoming UDP packets, resolves domain names using DoH (DNS over HTTPS) services, and supports IPv4 results. The server is designed to handle basic A record queries and can be configured using a JSON file.

## Features

- Listen for and respond to incoming UDP packets
- Resolve domain names using DoH services
- Support IPv4 results
- Handle basic A record queries
- Load configuration from a JSON file
- Custom domain resolution rules with wildcard support for subdomains

## Installation

1. Install [Go](https://golang.org/doc/install) (version 1.16 or later).
2. Clone the repository:

```sh
git clone https://github.com/yourusername/SimpleDNS.git
```

3. Change to the SimpleDNS directory:

```sh
cd SimpleDNS
```

4. Build the binary:

```sh
go build
```

## Usage

1. Edit the `config.json` file to configure the DoH URL and any custom domain resolution rules:

```json
{
  "dohurl": "https://dns.google.com/resolve",
  "rules": {
    "example.com": "0.0.0.0",
    "*.including-subdomain.com": "0.0.0.0"
  }
}
```

2. Run the SimpleDNS server:

```sh
./SimpleDNS
```

3. To test the server, you can use the `dig` command:

```sh
dig @localhost example.com
```

## Configuration

The configuration file is in JSON format and has two main properties:

- `dohurl`: The URL of the DoH service to use for domain name resolution (e.g., "https://dns.google.com/resolve").
- `rules`: A dictionary of domain names and their corresponding IPv4 addresses. You can specify wildcard subdomains using the "*.domain.com" syntax.

## License

This project is licensed under the [MIT License](LICENSE.md). 
