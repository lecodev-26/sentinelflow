-- Ensure an optional task reference belongs to the same tenant AND run as its
-- artifact/event. Tenant-only task foreign keys are not sufficient to enforce
-- the full execution lineage when two runs share a tenant.
ALTER TABLE v6_tasks
    ADD CONSTRAINT v6_tasks_tenant_run_id_id_key UNIQUE (tenant_id, run_id, id);

ALTER TABLE v6_artifacts
    ADD CONSTRAINT v6_artifacts_task_run_fk
    FOREIGN KEY (tenant_id, run_id, task_id)
    REFERENCES v6_tasks (tenant_id, run_id, id)
    ON DELETE CASCADE;

ALTER TABLE v6_execution_events
    ADD CONSTRAINT v6_execution_events_task_run_fk
    FOREIGN KEY (tenant_id, run_id, task_id)
    REFERENCES v6_tasks (tenant_id, run_id, id)
    ON DELETE CASCADE;
