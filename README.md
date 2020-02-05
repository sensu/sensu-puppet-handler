# sensu-puppet-handler

Deregister Sensu entities if they no longer have an associated Puppet node. The
puppet handler requires access to a SSL truststore and keystore, containing a
valid (and whitelisted) Puppet certificate, private key, and CA. The local
Puppet agent certificate, private key, and CA can be used.