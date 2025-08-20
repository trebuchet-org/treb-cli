package models

type ChangesetModels struct {
	Deployments      []*Deployment
	Transactions     []*Transaction
	SafeTransactions []*SafeTransaction
}

func (cm *ChangesetModels) HasChanges() bool {
	return (len(cm.Deployments) > 0 ||
		len(cm.Transactions) > 0 ||
		len(cm.SafeTransactions) > 0)
}

type Changeset struct {
	Create *ChangesetModels
	Update *ChangesetModels
	Delete *ChangesetModels
}

func (c *Changeset) HasChanges() bool {
	return (c.Create.HasChanges() ||
		c.Update.HasChanges() ||
		c.Delete.HasChanges())
}
