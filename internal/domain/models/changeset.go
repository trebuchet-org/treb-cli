package models

type ChangesetMetadata struct {
	Reasons map[string]string
}

type ChangesetModels struct {
	Deployments      []*Deployment
	Transactions     []*Transaction
	SafeTransactions []*SafeTransaction
	Metadata         ChangesetMetadata
}

func (cm *ChangesetModels) HasChanges() bool {
	return cm.Count() > 0
}

func (cm *ChangesetModels) Count() int {
	return len(cm.Deployments) + len(cm.Transactions) + len(cm.SafeTransactions)
}

type Changeset struct {
	Create ChangesetModels
	Update ChangesetModels
	Delete ChangesetModels
}

func (c *Changeset) HasChanges() bool {
	return c.Count() > 0
}

func (c *Changeset) Count() int {
	return c.Delete.Count() + c.Create.Count() + c.Update.Count()
}
