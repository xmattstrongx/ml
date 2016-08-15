package main

import sparta "github.com/mweagle/Sparta"

func main() {

	var lambdaFunctions []*sparta.LambdaAWSInfo
	createLambdaFn := GetCreateLambda()
	listLamdbaFn := GetListLambda()
	deleteLambdaFn := GetDeleteLambda()
	updateLambdaFn := GetUpdateLambda()
	emailLambdaFb := GetEmailLambda()

	lambdaFunctions = append(lambdaFunctions, createLambdaFn, listLamdbaFn, deleteLambdaFn, updateLambdaFn, emailLambdaFb)

	stage := sparta.NewStage("stage")
	apiGateway := sparta.NewAPIGateway("TaskListAPI", stage)

	// Bad framework wont allow me to assign multiple functions to a different resource and associate by HTTP verb.
	// resulting in awful RESTful route design... sigh...
	// Should be /api/tasks as route and then differentiate function calls based on verb
	// i.e POST -> createLambdaFn, GET -> listLamdbaFn
	apiGatewayCreateResource, _ := apiGateway.NewResource("/api/tasks/create", createLambdaFn)
	apiGatewayCreateResource.NewMethod("POST")

	apiGatewayListResource, _ := apiGateway.NewResource("/api/tasks/list", listLamdbaFn)
	apiGatewayListResource.NewMethod("GET")

	// Instead of asking user to pass in ID to delete in body this should be a parameter on the route.
	// Shouldnt be making excuses, but after battling this framework it has come to this...
	apiGatewayDeleteResource, _ := apiGateway.NewResource("/api/tasks/delete", deleteLambdaFn)
	apiGatewayDeleteResource.NewMethod("DELETE")

	apiGatewayUpdateResource, _ := apiGateway.NewResource("/api/tasks/update", updateLambdaFn)
	apiGatewayUpdateResource.NewMethod("PUT")

	// Deploy it
	sparta.Main("TaskLists",
		"TaskLists AWS Lambda functions",
		lambdaFunctions,
		apiGateway,
		nil)
}
