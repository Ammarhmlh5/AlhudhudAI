-- Migration 061: Add updated_at triggers
-- Description: Ensures updated_at is automatically updated on every record change.

-- 1. Create the update function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Apply to tenants
CREATE TRIGGER update_tenants_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 3. Apply to messages
CREATE TRIGGER update_messages_updated_at
BEFORE UPDATE ON messages
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 4. Apply to dispatch_channels
CREATE TRIGGER update_dispatch_channels_updated_at
BEFORE UPDATE ON dispatch_channels
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 5. Apply to incoming_messages
CREATE TRIGGER update_incoming_messages_updated_at
BEFORE UPDATE ON incoming_messages
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 6. Apply to accounting_transactions
CREATE TRIGGER update_accounting_transactions_updated_at
BEFORE UPDATE ON accounting_transactions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 7. Apply to dedicated_gateways
CREATE TRIGGER update_dedicated_gateways_updated_at
BEFORE UPDATE ON dedicated_gateways
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 8. Apply to gateway_statistics
CREATE TRIGGER update_gateway_statistics_updated_at
BEFORE UPDATE ON gateway_statistics
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
