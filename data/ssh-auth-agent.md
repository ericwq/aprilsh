# agent has no key, server has authorized_keys: ssh-ed25519

debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Next authentication method: publickey
debug1: Offering public key: /Users/qiwang/.ssh/id_rsa RSA SHA256:ba9XbxnmzpWpFIsItvvX5m1aQkOHyKJ9GEAXkeSBhdM explicit
debug2: we sent a publickey packet, wait for reply
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Trying private key: /Users/qiwang/.ssh/id_dsa
no such identity: /Users/qiwang/.ssh/id_dsa: No such file or directory
debug1: Trying private key: /Users/qiwang/.ssh/id_ecdsa
no such identity: /Users/qiwang/.ssh/id_ecdsa: No such file or directory
debug1: Offering public key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit
debug2: we sent a publickey packet, wait for reply
debug1: Server accepts key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit
Authenticated to localhost ([::1]:22) using "publickey".

# agent has rsa key, server has authorized_keys: ssh-ed25519

debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Next authentication method: publickey
debug1: Offering public key: /Users/qiwang/.ssh/id_rsa RSA SHA256:ba9XbxnmzpWpFIsItvvX5m1aQkOHyKJ9GEAXkeSBhdM explicit agent
debug2: we sent a publickey packet, wait for reply
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Trying private key: /Users/qiwang/.ssh/id_dsa
no such identity: /Users/qiwang/.ssh/id_dsa: No such file or directory
debug1: Trying private key: /Users/qiwang/.ssh/id_ecdsa
no such identity: /Users/qiwang/.ssh/id_ecdsa: No such file or directory
debug1: Offering public key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit
debug2: we sent a publickey packet, wait for reply
debug1: Server accepts key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit
Authenticated to localhost ([::1]:22) using "publickey".

# agent has two keys, server has authorized_keys: ssh-ed25519
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Next authentication method: publickey
debug1: Offering public key: /Users/qiwang/.ssh/id_rsa RSA SHA256:ba9XbxnmzpWpFIsItvvX5m1aQkOHyKJ9GEAXkeSBhdM explicit agent
debug2: we sent a publickey packet, wait for reply
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Offering public key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit agent
debug2: we sent a publickey packet, wait for reply
debug1: Server accepts key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit agent
Authenticated to localhost ([::1]:22) using "publickey".

# agent has two keys, server has no authorized_keys
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Next authentication method: publickey
debug1: Offering public key: /Users/qiwang/.ssh/id_rsa RSA SHA256:ba9XbxnmzpWpFIsItvvX5m1aQkOHyKJ9GEAXkeSBhdM explicit agent
debug2: we sent a publickey packet, wait for reply
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Offering public key: /Users/qiwang/.ssh/id_ed25519 ED25519 SHA256:Kf2WU+T3mJf9hk+aCmzLttqU84uV+DIvODOVLgZzb0k explicit agent
debug2: we sent a publickey packet, wait for reply
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug1: Trying private key: /Users/qiwang/.ssh/id_dsa
no such identity: /Users/qiwang/.ssh/id_dsa: No such file or directory
debug1: Trying private key: /Users/qiwang/.ssh/id_ecdsa
no such identity: /Users/qiwang/.ssh/id_ecdsa: No such file or directory
debug2: we did not send a packet, disable method
debug1: Next authentication method: keyboard-interactive
debug2: userauth_kbdint
debug2: we sent a keyboard-interactive packet, wait for reply
debug1: Authentications that can continue: publickey,password,keyboard-interactive
debug2: we did not send a packet, disable method
debug1: Next authentication method: password
