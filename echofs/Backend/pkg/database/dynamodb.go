package database

type DynamoDBRepo struct {
	Client *DynamoDB.Client
	TableName string
}

func NewDynamoDBRepo(clients *aws.Clients) *DynamoDBRepo {
	return &DynamoDBRepo{
		Client:    clients.DynamoDBClient,
		TableName: clients.AWSConfig.DynamoDBTableName,
	}
}