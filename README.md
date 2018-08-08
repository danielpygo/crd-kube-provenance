crd-kube-provenance

posting my work here for an aggregated server, used in extending kubernetes for maintaining provenance information of custom resources using the kubernetes audit feature

Working on doing query operators and making endpoints so that you can get provenance information by doing kubectl -raw=/some/endpoint/here/version

To run this, I will post the audit.yaml file, and some instructions for changes I made to the kubernetes source code to allow auditing.

