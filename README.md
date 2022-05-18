# Amesh

Amesh is a service mesh library for [Apache APISIX](http://apisix.apache.org/). It adapts the xDS protocol, receives data from Istio's control plane, and generates the APISIX data structure. In addition, APISIX does not use the traditional data center etcd, but uses shdict to exchange data with the Amesh library.

## Project Status

The Amesh project is currently under active development and is considered to be experimental.

## Architecture



## Quick Start

Refer to the document [Amesh Demo](./docs/en/demo.md) to quickly run the demo with Istio.

## How It Works

The Amesh injection template will create an init container, which will setup some iptables rules. These rules will capture all inbound and outbound traffic to APISIX, which allows APISIX to handle all traffic, providing the abilities and benefits of APISIX to full traffic management.

Configure Apache APISIX to use xDS as the `config_center`, which will make APISIX to load the Amesh library and load data from the shdict shared by APISIX and Amesh, instead of etcd.

The Amesh library will use the xDS protocol to communicate with the Istio control plane. It will translate xDS resources into APISIX routes and upstreams and store the data in shdict. When APISIX detects new data is written, the new configuration will be applied. In this way, APISIX can gain the ability of servise mesh by acting as sidecar of Istio.

## Support

If you encounter any problems with Amesh, please refer to our [Issues]((https://github.com/api7/amesh/issues)) list. If your issue doesn't appear in the issues list, please [create a new one](https://github.com/api7/amesh/issues/new).

## Contribute

## License

[Apache 2.0 LICENSE](./LICENSE)
