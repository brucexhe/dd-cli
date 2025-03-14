## dd-cli
A opensource docker deploy cli tool. Easy use to deploy a docker image to server like "dd deploy appapi".


## dd-server

run a dd server on your server: 
<pre>./dd-server</pre>
or
<pre>./dd-server -port 1688 </pre>

you will get a token after server start up successfully.
<pre>
Auth token:
e7a96fd2008111f0a2150242ac120003
</pre>


## dd-client
1. update config.yml 
2. update deploy.yml
3. run deploy command
<pre>dd deploy appapi</pre>

## local build

#### 1. install package
go get gopkg.in/yaml.v3

#### 2. build dd client
go build -o dd client.go

#### 3. build dd-server
go build -o dd-server server.go