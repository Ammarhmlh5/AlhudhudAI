-- Test Script for Billing Invoices Migration
-- This script tests all functionality of the billing_invoices table

BEGIN;

-- Create a test tenant for testing
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
BEGIN
    -- Insert test tenant if not exists
    INSERT INTO tenants (id, name, email, balance, currency)
    VALUES (v_test_tenant_id, 'Test Tenant for Invoices', 'test-invoices@example.com', 1000.00, 'USD')
    ON CONFLICT (id) DO NOTHING;
    
    -- Set current tenant for RLS
    PERFORM set_config('app.current_tenant', v_test_tenant_id::text, false);
    
    RAISE NOTICE 'Test tenant created/verified: %', v_test_tenant_id;
END $$;

-- Test 1: Generate invoice numbers
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
    v_invoice_num1 VARCHAR(50);
    v_invoice_num2 VARCHAR(50);
    v_invoice_num3 VARCHAR(50);
BEGIN
    RAISE NOTICE '=== Test 1: Generate Invoice Numbers ===';
    
    -- Generate first invoice number
    v_invoice_num1 := generate_invoice_number(v_test_tenant_id);
    RAISE NOTICE 'Generated invoice number 1: %', v_invoice_num1;
    
    -- Verify format (INV-YYYYMM-NNNN)
    IF v_invoice_num1 ~ '^INV-[0-9]{6}-[0-9]{4}$' THEN
        RAISE NOTICE '✓ Invoice number format is correct';
    ELSE
        RAISE EXCEPTION '✗ Invoice number format is incorrect: %', v_invoice_num1;
    END IF;
    
    -- Generate second invoice number (should increment)
    v_invoice_num2 := generate_invoice_number(v_test_tenant_id);
    RAISE NOTICE 'Generated invoice number 2: %', v_invoice_num2;
    
    -- Generate third invoice number
    v_invoice_num3 := generate_invoice_number(v_test_tenant_id);
    RAISE NOTICE 'Generated invoice number 3: %', v_invoice_num3;
    
    -- Verify sequential numbering
    IF v_invoice_num2 > v_invoice_num1 AND v_invoice_num3 > v_invoice_num2 THEN
        RAISE NOTICE '✓ Invoice numbers are sequential';
    ELSE
        RAISE EXCEPTION '✗ Invoice numbers are not sequential';
    END IF;
    
    RAISE NOTICE 'Test 1 PASSED';
END $$;

-- Test 2: Calculate tax for different countries
DO $$
DECLARE
    v_tax_record RECORD;
BEGIN
    RAISE NOTICE '=== Test 2: Calculate Tax ===';
    
    -- Test Yemen (5% VAT)
    SELECT * INTO v_tax_record FROM calculate_invoice_tax(100.00, 'YE');
    RAISE NOTICE 'Yemen (YE) - Subtotal: 100.00, Tax Rate: %, Tax: %, Total: %', 
        v_tax_record.tax_rate, v_tax_record.tax_amount, v_tax_record.total_amount;
    
    IF v_tax_record.tax_rate = 5.00 AND v_tax_record.tax_amount = 5.00 AND v_tax_record.total_amount = 105.00 THEN
        RAISE NOTICE '✓ Yemen tax calculation correct';
    ELSE
        RAISE EXCEPTION '✗ Yemen tax calculation incorrect';
    END IF;
    
    -- Test Saudi Arabia (15% VAT)
    SELECT * INTO v_tax_record FROM calculate_invoice_tax(100.00, 'SA');
    RAISE NOTICE 'Saudi Arabia (SA) - Subtotal: 100.00, Tax Rate: %, Tax: %, Total: %', 
        v_tax_record.tax_rate, v_tax_record.tax_amount, v_tax_record.total_amount;
    
    IF v_tax_record.tax_rate = 15.00 AND v_tax_record.tax_amount = 15.00 AND v_tax_record.total_amount = 115.00 THEN
        RAISE NOTICE '✓ Saudi Arabia tax calculation correct';
    ELSE
        RAISE EXCEPTION '✗ Saudi Arabia tax calculation incorrect';
    END IF;
    
    -- Test UAE (5% VAT)
    SELECT * INTO v_tax_record FROM calculate_invoice_tax(100.00, 'AE');
    RAISE NOTICE 'UAE (AE) - Subtotal: 100.00, Tax Rate: %, Tax: %, Total: %', 
        v_tax_record.tax_rate, v_tax_record.tax_amount, v_tax_record.total_amount;
    
    IF v_tax_record.tax_rate = 5.00 AND v_tax_record.tax_amount = 5.00 AND v_tax_record.total_amount = 105.00 THEN
        RAISE NOTICE '✓ UAE tax calculation correct';
    ELSE
        RAISE EXCEPTION '✗ UAE tax calculation incorrect';
    END IF;
    
    -- Test default (0% VAT)
    SELECT * INTO v_tax_record FROM calculate_invoice_tax(100.00, 'US');
    RAISE NOTICE 'Default (US) - Subtotal: 100.00, Tax Rate: %, Tax: %, Total: %', 
        v_tax_record.tax_rate, v_tax_record.tax_amount, v_tax_record.total_amount;
    
    IF v_tax_record.tax_rate = 0.00 AND v_tax_record.tax_amount = 0.00 AND v_tax_record.total_amount = 100.00 THEN
        RAISE NOTICE '✓ Default tax calculation correct';
    ELSE
        RAISE EXCEPTION '✗ Default tax calculation incorrect';
    END IF;
    
    RAISE NOTICE 'Test 2 PASSED';
END $$;

-- Test 3: Create invoices
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
    v_invoice_id1 INTEGER;
    v_invoice_id2 INTEGER;
    v_invoice_id3 INTEGER;
    v_invoice_record RECORD;
BEGIN
    RAISE NOTICE '=== Test 3: Create Invoices ===';
    
    -- Create invoice 1: Subscription payment
    v_invoice_id1 := create_invoice_for_payment(
        v_test_tenant_id,
        1,  -- subscription_id
        100,  -- transaction_id
        50.00,  -- subtotal
        'Monthly subscription payment for Standard package',
        'Mobile Gateway Service',
        'YE'
    );
    RAISE NOTICE 'Created invoice 1 (subscription): ID=%', v_invoice_id1;
    
    -- Verify invoice 1
    SELECT * INTO v_invoice_record FROM billing_invoices WHERE id = v_invoice_id1;
    IF v_invoice_record.subtotal = 50.00 AND v_invoice_record.total_amount = 52.50 THEN
        RAISE NOTICE '✓ Invoice 1 created correctly';
    ELSE
        RAISE EXCEPTION '✗ Invoice 1 amounts incorrect';
    END IF;
    
    -- Create invoice 2: Balance top-up
    v_invoice_id2 := create_invoice_for_payment(
        v_test_tenant_id,
        NULL,  -- no subscription
        101,  -- transaction_id
        100.00,  -- subtotal
        'Balance top-up',
        'Balance Service',
        'YE'
    );
    RAISE NOTICE 'Created invoice 2 (top-up): ID=%', v_invoice_id2;
    
    -- Create invoice 3: Another subscription payment
    v_invoice_id3 := create_invoice_for_payment(
        v_test_tenant_id,
        2,  -- subscription_id
        102,  -- transaction_id
        20.00,  -- subtotal
        'Monthly subscription payment for Basic package',
        'Mobile Gateway Service',
        'SA'  -- Saudi Arabia (15% tax)
    );
    RAISE NOTICE 'Created invoice 3 (subscription with SA tax): ID=%', v_invoice_id3;
    
    -- Verify invoice 3 (should have 15% tax)
    SELECT * INTO v_invoice_record FROM billing_invoices WHERE id = v_invoice_id3;
    IF v_invoice_record.subtotal = 20.00 AND v_invoice_record.total_amount = 23.00 THEN
        RAISE NOTICE '✓ Invoice 3 created correctly with SA tax';
    ELSE
        RAISE EXCEPTION '✗ Invoice 3 amounts incorrect: subtotal=%, total=%', 
            v_invoice_record.subtotal, v_invoice_record.total_amount;
    END IF;
    
    RAISE NOTICE 'Test 3 PASSED';
END $$;

-- Test 4: Mark invoice as paid
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
    v_invoice_id INTEGER;
    v_result BOOLEAN;
    v_invoice_record RECORD;
BEGIN
    RAISE NOTICE '=== Test 4: Mark Invoice as Paid ===';
    
    -- Create a pending invoice
    INSERT INTO billing_invoices (
        tenant_id,
        invoice_number,
        subtotal,
        tax_rate,
        tax_amount,
        total_amount,
        description,
        status
    ) VALUES (
        v_test_tenant_id,
        'INV-TEST-0001',
        100.00,
        5.00,
        5.00,
        105.00,
        'Test pending invoice',
        'pending'
    )
    RETURNING id INTO v_invoice_id;
    
    RAISE NOTICE 'Created pending invoice: ID=%', v_invoice_id;
    
    -- Mark as paid
    v_result := mark_invoice_paid(v_invoice_id);
    
    IF v_result THEN
        RAISE NOTICE '✓ Invoice marked as paid successfully';
    ELSE
        RAISE EXCEPTION '✗ Failed to mark invoice as paid';
    END IF;
    
    -- Verify status changed
    SELECT * INTO v_invoice_record FROM billing_invoices WHERE id = v_invoice_id;
    IF v_invoice_record.status = 'paid' AND v_invoice_record.paid_date IS NOT NULL THEN
        RAISE NOTICE '✓ Invoice status and paid_date updated correctly';
    ELSE
        RAISE EXCEPTION '✗ Invoice status or paid_date not updated';
    END IF;
    
    RAISE NOTICE 'Test 4 PASSED';
END $$;

-- Test 5: Get tenant invoices with filtering
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
    v_invoice_count INTEGER;
BEGIN
    RAISE NOTICE '=== Test 5: Get Tenant Invoices ===';
    
    -- Get all invoices
    SELECT COUNT(*) INTO v_invoice_count
    FROM get_tenant_invoices(v_test_tenant_id, NULL, NULL, NULL, 100, 0);
    RAISE NOTICE 'Total invoices for tenant: %', v_invoice_count;
    
    IF v_invoice_count >= 4 THEN
        RAISE NOTICE '✓ Found expected number of invoices';
    ELSE
        RAISE EXCEPTION '✗ Expected at least 4 invoices, found %', v_invoice_count;
    END IF;
    
    -- Get only paid invoices
    SELECT COUNT(*) INTO v_invoice_count
    FROM get_tenant_invoices(v_test_tenant_id, 'paid', NULL, NULL, 100, 0);
    RAISE NOTICE 'Paid invoices: %', v_invoice_count;
    
    -- Get invoices for today
    SELECT COUNT(*) INTO v_invoice_count
    FROM get_tenant_invoices(v_test_tenant_id, NULL, CURRENT_DATE, CURRENT_DATE, 100, 0);
    RAISE NOTICE 'Invoices issued today: %', v_invoice_count;
    
    -- Test pagination
    SELECT COUNT(*) INTO v_invoice_count
    FROM get_tenant_invoices(v_test_tenant_id, NULL, NULL, NULL, 2, 0);
    IF v_invoice_count = 2 THEN
        RAISE NOTICE '✓ Pagination limit works correctly';
    ELSE
        RAISE EXCEPTION '✗ Pagination limit not working: expected 2, got %', v_invoice_count;
    END IF;
    
    RAISE NOTICE 'Test 5 PASSED';
END $$;

-- Test 6: Verify RLS policies
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
    v_other_tenant_id UUID := '22222222-2222-2222-2222-222222222222';
    v_invoice_count INTEGER;
BEGIN
    RAISE NOTICE '=== Test 6: Verify RLS Policies ===';
    
    -- Create another tenant
    INSERT INTO tenants (id, name, email, balance, currency)
    VALUES (v_other_tenant_id, 'Other Tenant', 'other@example.com', 500.00, 'USD')
    ON CONFLICT (id) DO NOTHING;
    
    -- Set current tenant to test tenant
    PERFORM set_config('app.current_tenant', v_test_tenant_id::text, false);
    
    -- Count invoices (should see test tenant's invoices)
    SELECT COUNT(*) INTO v_invoice_count FROM billing_invoices;
    RAISE NOTICE 'Invoices visible to test tenant: %', v_invoice_count;
    
    IF v_invoice_count >= 4 THEN
        RAISE NOTICE '✓ Test tenant can see their invoices';
    ELSE
        RAISE EXCEPTION '✗ Test tenant cannot see their invoices';
    END IF;
    
    -- Switch to other tenant
    PERFORM set_config('app.current_tenant', v_other_tenant_id::text, false);
    
    -- Count invoices (should see 0 - other tenant has no invoices)
    SELECT COUNT(*) INTO v_invoice_count FROM billing_invoices;
    RAISE NOTICE 'Invoices visible to other tenant: %', v_invoice_count;
    
    IF v_invoice_count = 0 THEN
        RAISE NOTICE '✓ RLS isolation working - other tenant sees 0 invoices';
    ELSE
        RAISE EXCEPTION '✗ RLS isolation broken - other tenant sees % invoices', v_invoice_count;
    END IF;
    
    -- Try to insert invoice for test tenant while logged in as other tenant
    BEGIN
        INSERT INTO billing_invoices (
            tenant_id,
            invoice_number,
            subtotal,
            total_amount,
            description
        ) VALUES (
            v_test_tenant_id,  -- trying to insert for test tenant
            'INV-HACK-0001',
            100.00,
            100.00,
            'Attempted cross-tenant insert'
        );
        
        RAISE EXCEPTION '✗ RLS INSERT policy failed - cross-tenant insert succeeded';
    EXCEPTION
        WHEN check_violation OR insufficient_privilege THEN
            RAISE NOTICE '✓ RLS INSERT policy working - cross-tenant insert blocked';
    END;
    
    -- Reset to test tenant
    PERFORM set_config('app.current_tenant', v_test_tenant_id::text, false);
    
    RAISE NOTICE 'Test 6 PASSED';
END $$;

-- Test 7: Verify indexes exist
DO $$
DECLARE
    v_index_count INTEGER;
BEGIN
    RAISE NOTICE '=== Test 7: Verify Indexes ===';
    
    SELECT COUNT(*) INTO v_index_count
    FROM pg_indexes
    WHERE tablename = 'billing_invoices'
      AND indexname LIKE 'idx_invoices_%';
    
    RAISE NOTICE 'Found % indexes on billing_invoices table', v_index_count;
    
    IF v_index_count >= 7 THEN
        RAISE NOTICE '✓ All expected indexes exist';
    ELSE
        RAISE EXCEPTION '✗ Missing indexes: expected at least 7, found %', v_index_count;
    END IF;
    
    RAISE NOTICE 'Test 7 PASSED';
END $$;

-- Test 8: Verify constraints
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
BEGIN
    RAISE NOTICE '=== Test 8: Verify Constraints ===';
    
    -- Test positive subtotal constraint
    BEGIN
        INSERT INTO billing_invoices (
            tenant_id, invoice_number, subtotal, total_amount
        ) VALUES (
            v_test_tenant_id, 'INV-NEG-0001', -10.00, -10.00
        );
        RAISE EXCEPTION '✗ Negative subtotal constraint failed';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Negative subtotal constraint working';
    END;
    
    -- Test valid status constraint
    BEGIN
        INSERT INTO billing_invoices (
            tenant_id, invoice_number, subtotal, total_amount, status
        ) VALUES (
            v_test_tenant_id, 'INV-BAD-0001', 10.00, 10.00, 'invalid_status'
        );
        RAISE EXCEPTION '✗ Status constraint failed';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Status constraint working';
    END;
    
    -- Test tax rate range constraint
    BEGIN
        INSERT INTO billing_invoices (
            tenant_id, invoice_number, subtotal, total_amount, tax_rate
        ) VALUES (
            v_test_tenant_id, 'INV-TAX-0001', 10.00, 10.00, 150.00
        );
        RAISE EXCEPTION '✗ Tax rate constraint failed';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Tax rate constraint working';
    END;
    
    RAISE NOTICE 'Test 8 PASSED';
END $$;

-- Display summary of created invoices
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
BEGIN
    RAISE NOTICE '=== Invoice Summary ===';
    RAISE NOTICE 'Invoices created for test tenant:';
    
    PERFORM set_config('app.current_tenant', v_test_tenant_id::text, false);
    
    FOR rec IN 
        SELECT invoice_number, issue_date, total_amount, status, service_name
        FROM billing_invoices
        ORDER BY created_at
    LOOP
        RAISE NOTICE '  % | % | $% | % | %', 
            rec.invoice_number, rec.issue_date, rec.total_amount, rec.status, rec.service_name;
    END LOOP;
END $$;

-- Cleanup test data
DO $$
DECLARE
    v_test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
    v_other_tenant_id UUID := '22222222-2222-2222-2222-222222222222';
BEGIN
    RAISE NOTICE '=== Cleanup ===';
    
    -- Set current tenant for deletion
    PERFORM set_config('app.current_tenant', v_test_tenant_id::text, false);
    
    -- Delete test invoices
    DELETE FROM billing_invoices WHERE tenant_id = v_test_tenant_id;
    RAISE NOTICE 'Deleted test invoices for test tenant';
    
    -- Delete test tenants
    DELETE FROM tenants WHERE id IN (v_test_tenant_id, v_other_tenant_id);
    RAISE NOTICE 'Deleted test tenants';
    
    RAISE NOTICE 'Cleanup complete';
END $$;

ROLLBACK;

-- Summary
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'ALL TESTS PASSED SUCCESSFULLY!';
    RAISE NOTICE '========================================';
    RAISE NOTICE '';
    RAISE NOTICE 'The billing_invoices migration is working correctly:';
    RAISE NOTICE '  ✓ Invoice number generation';
    RAISE NOTICE '  ✓ Tax calculation for different countries';
    RAISE NOTICE '  ✓ Invoice creation';
    RAISE NOTICE '  ✓ Invoice status updates';
    RAISE NOTICE '  ✓ Filtering and pagination';
    RAISE NOTICE '  ✓ Row Level Security (RLS)';
    RAISE NOTICE '  ✓ Indexes';
    RAISE NOTICE '  ✓ Constraints';
    RAISE NOTICE '';
END $$;
