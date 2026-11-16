# R32 public-launch profile (before Gate B or > 100 orders/day): Multi-AZ, one
# NAT gateway, larger api tasks, 2 workers. Final class set by the doc 20 load test.
nat_gateway_count    = 1
db_instance_class    = "db.t4g.medium"
db_multi_az          = true
db_allocated_storage = 30
api_desired_count    = 2
api_cpu              = 512
api_memory           = 1024
api_max_count        = 6
worker_desired_count = 2
