# DDNS Project

This project is a simple implementation of a Dynamic DNS (DDNS) service using Go. It allows users to create and update subdomain entries with their current IP address.

## Features

- Create a new subdomain and associate it with your current IP address
- Update an existing subdomain with a new IP address
- Automatically redirect visitors to the correct IP address when accessing a subdomain

## Prerequisites

- Go installed on your machine
- A database system compatible with the `database` package

## Installation

1. Clone the repository:

git clone https://github.com/SassoF/ddns.git


### Create a new subdomain

To create a new subdomain, send a GET request to `/newDomain` with the `domain` query parameter:
http://yourDomain.com/newDomain?domain=example
This will associate the subdomain `example` with your current IP address.

### Update a subdomain

To update an existing subdomain, send a GET request to `/update` with the `domain`, `token`, and optionally `ip` query parameters:
http://yourDomain.com/update?domain=example&token=abc123&ip=1.2.3.4
This will update the subdomain `example` with the specified IP address (`1.2.3.4` in this case). If the `ip` parameter is omitted, it will use your current IP address.

### Access a subdomain

To access a subdomain, simply visit the subdomain URL in your browser:
http://example.yourDomain.com
If the subdomain exists, you will be redirected to the associated IP address.
