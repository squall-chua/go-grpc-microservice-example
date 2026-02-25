package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	itemsCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "items_created_total",
		Help: "The total number of successfully created items",
	})

	itemsUpdatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "items_updated_total",
		Help: "The total number of successfully updated items",
	})

	itemsDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "items_deleted_total",
		Help: "The total number of successfully deleted items",
	})
)
