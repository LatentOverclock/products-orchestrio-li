# project.v1 - Initial requirements (delta)

## Goal
Build a product management application.

## Functional requirements
1. A product has the following properties:
   - name
   - purchase-link
   - shop-link
   - booqable-link
   - manual-link
   - inspection-link
   - description (text field)
   - status
2. Product status must be one of:
   - `mafo`
   - `write-manual`
   - `all-done`
3. Users can create, edit, and delete products.
4. Users can view products grouped by status (kanban style).
5. Users can view products in a table-style list.
6. Users can click a product and open a product detail view page listing all product details.

## Non-functional requirements
7. Keep deployment-specific host/domain configuration outside application code.
8. Validate each project requirement with automated tests.
