# R32 closed-pilot profile: RDS Single-AZ db.t4g.small, 2 small api + 1 worker,
# no NAT (public subnets + compensating controls, R28), no collector gateway.
nat_gateway_count    = 0
db_instance_class    = "db.t4g.small"
db_multi_az          = false
db_allocated_storage = 20
api_desired_count    = 2
api_cpu              = 256
api_memory           = 512
api_max_count        = 4
worker_desired_count = 1
