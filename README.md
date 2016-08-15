# TaskList API

### Framework Used: http://gosparta.io/
  - Sparta was chosen on the promise and ability to quickly deploy AWS lambda FaaS in GO
  - Towards the end of developer issues were found with this framework.

### Sparta Framework Issues: 
  - No ability to add .json, .yaml etc. files to be pacakged with deployment // no config.json :(
  - API Gateway generation only allows for one function per route // inability to deploy API where routes determine function :(

### Sparta Command recap:
- ```go run *.go provision --s3Bucket $S3_BUCKET``` // provisions and deploys lambdas, API gateways etc
- ```go run *.go delete``` // deletes lambdas, apis, etc

### The bad.
- No tests yet
- Hardcoded connection strings because cant get framework to include my config.json unless I manually drop it
- Not returning 400 for bad requests even though response informs user request is bad.

### IAM Roles
- taskAccessRole
- emailRole


### Todos

 - Write Tests, tests and more test!
 - Correct HTTP status codes for returns (particularly 400 bad request)
 - Refactor all DB interaction into a repo package
 - Add swagger(OpenAPI) definition
 - Add More Code Comments
 - Watch Sparta.io die in a fire
 - Learn Node.JS so I dont need Sparta.io after the fire incident