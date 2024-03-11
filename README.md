# Prerequisite

- it's required to have ssh-agent running
```bash
eval $(ssh-agent)
```
- it's required to have private key registered in ssh agent
```bash
ssh-add ~/.ssh/private_key
```

# Testing
External dependencies:
- GitHub API
  - Can be mocked with fake httpclient or by abstracting client with interface
  - Can use the actual API, but need to find a way for garbage collection
- git push
  - there are present git server for mocking (FGS). But it may be problematic, because right now we are trying to derive
    github org and repo name from origin url
  - We could use real github repo
- git local repository. operations with FS
  - Can be mocked with virtual FS 
  - Can use temporary folders for testing
- template render. operations with FS
  - API doesn't allow to mock with virtual FS
  - Can use temporary folders for testing