# DPCTL - Developer Platform CTL
A CLI tool for providing high-level interactions for fellow developers.

# Usage 

Before start using the CLI you should initialize it.
For the initialization you have to provide:
- initialization file: [dpctl-init-example.yaml](dpctl-init-example.yaml)
- your person GitHub token to perform operations on your behalf

After the initialization you can start using `dpctl`. 

To check for available operations run:
```bash 
dpctl --help
```

## Note
At the moment the CLI expects that you use SSH as the protocol for `git remote`.

Because of this it's expected that:
- you have ssh-agent running
```bash
eval $(ssh-agent)
```
- you have private key registered in ssh agent and in GitHub
```bash
ssh-add ~/.ssh/private_key
```