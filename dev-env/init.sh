mkdir -p ~/repos
mkdir ~/repos/dplatform-envs-test

eval $(ssh-agent)
grep -slR "PRIVATE" ~/.ssh/ | xargs ssh-add