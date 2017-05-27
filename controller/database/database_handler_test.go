package database_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	"code.cloudfoundry.org/silk/controller"
	"code.cloudfoundry.org/silk/controller/database"
	"code.cloudfoundry.org/silk/controller/database/fakes"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	migrate "github.com/rubenv/sql-migrate"
)

var _ = Describe("DatabaseHandler", func() {
	var (
		databaseHandler    *database.DatabaseHandler
		realDb             *sqlx.DB
		realMigrateAdapter *database.MigrateAdapter
		dbConfig           db.Config
		mockDb             *fakes.Db
		mockMigrateAdapter *fakes.MigrateAdapter
		lease              controller.Lease
		lease2             controller.Lease
		timeout            time.Duration
		mockDeleter        *fakes.SubnetDeleter
		realDeleter        database.SubnetDeleter
	)
	BeforeEach(func() {
		mockDb = &fakes.Db{}
		mockMigrateAdapter = &fakes.MigrateAdapter{}

		dbConfig = testsupport.GetDBConfig()
		dbConfig.DatabaseName = fmt.Sprintf("test_db_%03d_%x", GinkgoParallelNode(), rand.Int())
		testsupport.CreateDatabase(dbConfig)

		timeout = 2 * time.Second

		mockDeleter = &fakes.SubnetDeleter{}
		realDeleter = &database.Deleter{}

		var err error
		realDb, err = db.GetConnectionPool(dbConfig)
		Expect(err).NotTo(HaveOccurred())

		realMigrateAdapter = &database.MigrateAdapter{}

		mockDb.DriverNameReturns(realDb.DriverName())
		lease = controller.Lease{
			UnderlayIP:          "10.244.11.22",
			OverlaySubnet:       "10.255.17.0/24",
			OverlayHardwareAddr: "ee:ee:0a:ff:11:00",
		}
		lease2 = controller.Lease{
			UnderlayIP:          "10.244.22.33",
			OverlaySubnet:       "10.255.93.15/32",
			OverlayHardwareAddr: "ee:ee:0a:ff:5d:0f",
		}
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		Expect(testsupport.RemoveDatabase(dbConfig)).To(Succeed())
	})

	Describe("Migrate", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
			mockMigrateAdapter.ExecReturns(43, nil)
		})
		It("calls the migrate adapter", func() {
			By("returning the results from the migrator")
			numMigrations, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			Expect(numMigrations).To(Equal(43))

			By("calling the migrator")
			Expect(mockMigrateAdapter.ExecCallCount()).To(Equal(1))
			db, dbType, migrations, dir := mockMigrateAdapter.ExecArgsForCall(0)
			Expect(db).To(Equal(mockDb))
			Expect(dbType).To(Equal(realDb.DriverName()))
			if dbType == "postgres" {
				Expect(migrations).To(Equal(migrate.MemoryMigrationSource{
					Migrations: []*migrate.Migration{
						&migrate.Migration{
							Id:   "1",
							Up:   []string{"CREATE TABLE IF NOT EXISTS subnets (id SERIAL PRIMARY KEY, underlay_ip varchar(15) NOT NULL, overlay_subnet varchar(18) NOT NULL, overlay_hwaddr varchar(17) NOT NULL, last_renewed_at bigint NOT NULL, UNIQUE (underlay_ip), UNIQUE (overlay_subnet), UNIQUE (overlay_hwaddr));"},
							Down: []string{"DROP TABLE subnets"},
						},
					},
				}))
			} else {
				Expect(migrations).To(Equal(migrate.MemoryMigrationSource{
					Migrations: []*migrate.Migration{
						&migrate.Migration{
							Id:   "1",
							Up:   []string{"CREATE TABLE IF NOT EXISTS subnets (id int NOT NULL AUTO_INCREMENT, PRIMARY KEY (id), underlay_ip varchar(15) NOT NULL, overlay_subnet varchar(18) NOT NULL, overlay_hwaddr varchar(17) NOT NULL, last_renewed_at bigint NOT NULL, UNIQUE (underlay_ip), UNIQUE (overlay_subnet), UNIQUE (overlay_hwaddr));"},
							Down: []string{"DROP TABLE subnets"},
						},
					},
				}))
			}
			Expect(dir).To(Equal(migrate.Up))
		})
		Context("when the migrator fails", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockMigrateAdapter.ExecReturns(0, errors.New("guava"))
			})
			It("returns the error", func() {
				_, err := databaseHandler.Migrate()
				Expect(err).To(MatchError("migrating: guava"))
			})
		})
	})

	Describe("AddEntry", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
			mockDb.ExecReturns(nil, nil)
		})

		Context("when the database type is postgres", func() {
			BeforeEach(func() {
				mockDb.DriverNameReturns("postgres")
			})
			It("adds an entry to the DB", func() {
				err := databaseHandler.AddEntry(lease)
				Expect(err).NotTo(HaveOccurred())

				Expect(mockDb.ExecContextCallCount()).To(Equal(1))
				_, query, options := mockDb.ExecContextArgsForCall(0)
				Expect(query).To(Equal("INSERT INTO subnets (underlay_ip, overlay_subnet, overlay_hwaddr, last_renewed_at) VALUES ('10.244.11.22', '10.255.17.0/24', 'ee:ee:0a:ff:11:00', EXTRACT(EPOCH FROM now())::numeric::integer)"))
				Expect(options).To(BeNil())
			})
		})

		Context("when the database type is mysql", func() {
			BeforeEach(func() {
				mockDb.DriverNameReturns("mysql")
			})
			It("adds an entry to the DB", func() {
				err := databaseHandler.AddEntry(lease)
				Expect(err).NotTo(HaveOccurred())

				Expect(mockDb.ExecContextCallCount()).To(Equal(1))
				_, query, options := mockDb.ExecContextArgsForCall(0)
				Expect(query).To(Equal("INSERT INTO subnets (underlay_ip, overlay_subnet, overlay_hwaddr, last_renewed_at) VALUES ('10.244.11.22', '10.255.17.0/24', 'ee:ee:0a:ff:11:00', UNIX_TIMESTAMP())"))
				Expect(options).To(BeNil())
			})
		})

		Context("when the database type is not supported", func() {
			BeforeEach(func() {
				mockDb.DriverNameReturns("foo")
			})
			It("returns an error", func() {
				err := databaseHandler.RenewLeaseForUnderlayIP("1.2.3.4")
				Expect(err).To(MatchError("database type foo is not supported"))
			})
		})

		Context("when the database exec returns an error", func() {
			BeforeEach(func() {
				mockDb.ExecContextReturns(nil, errors.New("apple"))
			})
			It("returns a sensible error", func() {
				err := databaseHandler.AddEntry(lease)
				Expect(err).To(MatchError("adding entry: apple"))
			})
		})
	})

	Describe("DeleteEntry", func() {
		var realTx *sqlx.Tx
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, realDeleter, timeout)
			_, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease)
			Expect(err).NotTo(HaveOccurred())

			By("checking that the lease is present")
			leases, err := databaseHandler.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(leases).To(ContainElement(lease))

			realTx, err = realDb.BeginTxx(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes an entry from the DB", func() {
			err := databaseHandler.DeleteEntry("10.244.11.22")
			Expect(err).NotTo(HaveOccurred())

			By("checking that the lease is not present")
			leases, err := databaseHandler.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(leases).NotTo(ContainElement(lease))
		})

		Context("when starting the transaction fails", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.BeginTxxReturns(nil, errors.New("celery"))
			})
			It("returns a sensible error", func() {
				err := databaseHandler.DeleteEntry("some-underlay")
				Expect(err).To(MatchError("begin transaction: celery"))
			})
		})

		Context("when subnet delete fails", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.BeginTxxReturns(realTx, nil)
				mockDeleter.DeleteReturns(nil, errors.New("banana"))
			})
			It("returns a sensible error", func() {
				err := databaseHandler.DeleteEntry("some-underlay")
				Expect(err).To(MatchError("deleting entry: banana"))

				Expect(mockDeleter.DeleteCallCount()).To(Equal(1))
				tx, underlayIP, time := mockDeleter.DeleteArgsForCall(0)
				Expect(tx).To(Equal(realTx))
				Expect(underlayIP).To(Equal("some-underlay"))
				Expect(time).To(Equal(timeout))
			})
		})

		Context("when parsing the result fails", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.BeginTxxReturns(realTx, nil)
				badResult := &fakes.SqlResult{}
				badResult.RowsAffectedReturns(0, errors.New("potato"))
				mockDeleter.DeleteReturns(badResult, nil)
			})

			It("returns an error", func() {
				err := databaseHandler.DeleteEntry("10.244.11.22")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("parse result: potato"))
			})
		})

		Context("when no entry exists", func() {
			It("returns a RecordNotAffectedError", func() {
				err := databaseHandler.DeleteEntry("8.8.8.8")
				Expect(err).To(Equal(database.RecordNotAffectedError))
			})
		})

		Context("when more than one row in result", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.BeginTxxReturns(realTx, nil)
				badResult := &fakes.SqlResult{}
				badResult.RowsAffectedReturns(2, nil)
				mockDeleter.DeleteReturns(badResult, nil)
			})

			It("returns a MultipleRecordsAffectedError", func() {
				err := databaseHandler.DeleteEntry("10.244.11.22")
				Expect(err).To(Equal(database.MultipleRecordsAffectedError))
			})
		})
	})

	Describe("LeaseForUnderlayIP", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, mockDeleter, timeout)
			_, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease)
			Expect(err).NotTo(HaveOccurred())
		})
		It("returns the subnet for the given underlay IP", func() {
			found, err := databaseHandler.LeaseForUnderlayIP("10.244.11.22")
			Expect(err).NotTo(HaveOccurred())
			Expect(*found).To(Equal(lease))
		})

		Context("when there is no entry for the underlay ip", func() {
			It("returns nil", func() {
				entry, err := databaseHandler.LeaseForUnderlayIP("10.244.11.23")
				Expect(err).NotTo(HaveOccurred())
				Expect(entry).To(BeNil())
			})
		})
	})

	Describe("RenewLeaseForUnderlayIP", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
		})

		Context("when the database is postgres", func() {
			BeforeEach(func() {
				mockDb.DriverNameReturns("postgres")
			})
			It("updates the last renewed at time", func() {
				err := databaseHandler.RenewLeaseForUnderlayIP("1.2.3.4")
				Expect(err).NotTo(HaveOccurred())

				Expect(mockDb.ExecContextCallCount()).To(Equal(1))
				_, query, options := mockDb.ExecContextArgsForCall(0)
				Expect(query).To(Equal("UPDATE subnets SET last_renewed_at = EXTRACT(EPOCH FROM now())::numeric::integer WHERE underlay_ip = '1.2.3.4'"))
				Expect(options).To(BeNil())
			})
		})

		Context("when the database is mysql", func() {
			BeforeEach(func() {
				mockDb.DriverNameReturns("mysql")
			})
			It("updates the last renewed at time", func() {
				err := databaseHandler.RenewLeaseForUnderlayIP("1.2.3.4")
				Expect(err).NotTo(HaveOccurred())

				Expect(mockDb.ExecContextCallCount()).To(Equal(1))
				_, query, options := mockDb.ExecContextArgsForCall(0)
				Expect(query).To(Equal("UPDATE subnets SET last_renewed_at = UNIX_TIMESTAMP() WHERE underlay_ip = '1.2.3.4'"))
				Expect(options).To(BeNil())
			})
		})

		Context("when the database type is not supported", func() {
			BeforeEach(func() {
				mockDb.DriverNameReturns("foo")
			})
			It("returns an error", func() {
				err := databaseHandler.RenewLeaseForUnderlayIP("1.2.3.4")
				Expect(err).To(MatchError("database type foo is not supported"))
			})
		})

		Context("when the database exec returns an error", func() {
			BeforeEach(func() {
				mockDb.ExecContextReturns(nil, errors.New("apple"))
			})
			It("returns a sensible error", func() {
				err := databaseHandler.RenewLeaseForUnderlayIP("1.2.3.4")
				Expect(err).To(MatchError("renewing lease: apple"))
			})
		})
	})

	Describe("LastRenewedAtForUnderlayIP", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, mockDeleter, timeout)
			_, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease)
			Expect(err).NotTo(HaveOccurred())
		})
		It("selects the last_renewed_at time for the lease", func() {
			lastRenewedAt, err := databaseHandler.LastRenewedAtForUnderlayIP("10.244.11.22")
			Expect(err).NotTo(HaveOccurred())
			Expect(lastRenewedAt).To(BeNumerically(">", 0))
		})
		It("gets updated when the lease is renewed", func() {
			createdAt, err := databaseHandler.LastRenewedAtForUnderlayIP("10.244.11.22")
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(1 * time.Second)
			err = databaseHandler.RenewLeaseForUnderlayIP("10.244.11.22")
			Expect(err).NotTo(HaveOccurred())
			updatedAt, err := databaseHandler.LastRenewedAtForUnderlayIP("10.244.11.22")
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedAt).To(BeNumerically(">", createdAt))
		})

		Context("when there is no entry for the underlay ip", func() {
			It("returns an error", func() {
				_, err := databaseHandler.LastRenewedAtForUnderlayIP("10.244.11.23")
				Expect(err).To(MatchError("sql: no rows in result set"))
			})
		})
	})

	Describe("All", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, mockDeleter, timeout)
			_, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease)
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease2)
			Expect(err).NotTo(HaveOccurred())
		})
		It("all the saved subnets", func() {
			leases, err := databaseHandler.All()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(leases)).To(Equal(2))
			Expect(leases).To(ConsistOf([]controller.Lease{
				{
					UnderlayIP:          "10.244.11.22",
					OverlaySubnet:       "10.255.17.0/24",
					OverlayHardwareAddr: "ee:ee:0a:ff:11:00",
				},
				{
					UnderlayIP:          "10.244.22.33",
					OverlaySubnet:       "10.255.93.15/32",
					OverlayHardwareAddr: "ee:ee:0a:ff:5d:0f",
				},
			}))
		})

		Context("when the query fails", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.QueryContextReturns(nil, errors.New("strawberry"))
			})
			It("returns an error", func() {
				_, err := databaseHandler.All()
				Expect(err).To(MatchError("selecting all subnets: strawberry"))
			})
		})

		Context("when the parsing the result fails", func() {
			var rows *sql.Rows
			BeforeEach(func() {
				var err error
				rows, err = realDb.Query("SELECT 1")
				Expect(err).NotTo(HaveOccurred())

				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.QueryContextReturns(rows, nil)
			})

			AfterEach(func() {
				Expect(rows.Close()).To(Succeed())
			})

			It("returns an error", func() {
				_, err := databaseHandler.All()
				Expect(err.Error()).To(ContainSubstring("parsing result for all subnets"))
			})
		})
	})

	Describe("AllActive", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, mockDeleter, timeout)
			_, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease)
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the leases which have been renewed within the expiration time", func() {
			leases, err := databaseHandler.AllActive(1000)
			Expect(err).NotTo(HaveOccurred())

			Expect(leases).To(HaveLen(2))
			Expect(leases).To(ConsistOf([]controller.Lease{
				{
					UnderlayIP:          "10.244.11.22",
					OverlaySubnet:       "10.255.17.0/24",
					OverlayHardwareAddr: "ee:ee:0a:ff:11:00",
				},
				{
					UnderlayIP:          "10.244.22.33",
					OverlaySubnet:       "10.255.93.15/32",
					OverlayHardwareAddr: "ee:ee:0a:ff:5d:0f",
				},
			}))

			leases, err = databaseHandler.AllActive(0)
			Expect(err).NotTo(HaveOccurred())

			Expect(leases).To(HaveLen(0))
		})

		Context("when the db driver name is not supported", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.DriverNameReturns("foo")
			})
			It("should return an error", func() {
				_, err := databaseHandler.AllActive(1000)
				Expect(err).To(MatchError("database type foo is not supported"))
			})
		})

		Context("when the query fails", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.QueryContextReturns(nil, errors.New("strawberry"))
			})
			It("returns an error", func() {
				_, err := databaseHandler.AllActive(100)
				Expect(err).To(MatchError("selecting all active subnets: strawberry"))
			})
		})

		Context("when the parsing the result fails", func() {
			var rows *sql.Rows
			BeforeEach(func() {
				var err error
				rows, err = realDb.Query("SELECT 1")
				Expect(err).NotTo(HaveOccurred())

				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.QueryContextReturns(rows, nil)
			})

			AfterEach(func() {
				Expect(rows.Close()).To(Succeed())
			})

			It("returns an error", func() {
				_, err := databaseHandler.AllActive(100)
				Expect(err.Error()).To(ContainSubstring("parsing result for all active subnets"))
			})
		})
	})

	Describe("OldestExpired", func() {
		BeforeEach(func() {
			databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, mockDeleter, timeout)
			_, err := databaseHandler.Migrate()
			Expect(err).NotTo(HaveOccurred())
			err = databaseHandler.AddEntry(lease)
			Expect(err).NotTo(HaveOccurred())
		})

		It("gets the oldest lease that is expired", func() {
			expiredLease, err := databaseHandler.OldestExpired(0)
			Expect(err).NotTo(HaveOccurred())

			Expect(expiredLease).To(Equal(&lease))
		})

		Context("when the database type is not supported", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
				mockDb.DriverNameReturns("foo")
			})
			It("returns an error", func() {
				_, err := databaseHandler.OldestExpired(23)
				Expect(err).To(MatchError("database type foo is not supported"))
			})
		})

		Context("when no lease is expired", func() {
			BeforeEach(func() {
				databaseHandler = database.NewDatabaseHandler(realMigrateAdapter, realDb, mockDeleter, timeout)
				_, err := databaseHandler.Migrate()
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns nil and does not error", func() {
				lease, err := databaseHandler.OldestExpired(23)
				Expect(err).NotTo(HaveOccurred())
				Expect(lease).To(BeNil())
			})
		})

		Context("when parsing the result fails", func() {
			var result *sql.Row
			BeforeEach(func() {
				result = realDb.QueryRow("SELECT 1")

				mockDb.QueryRowContextReturns(result)
				databaseHandler = database.NewDatabaseHandler(mockMigrateAdapter, mockDb, mockDeleter, timeout)
			})
			It("returns an error", func() {
				_, err := databaseHandler.OldestExpired(23)
				Expect(err).To(MatchError(ContainSubstring("scan result:")))
			})
		})
	})
})

//go:generate counterfeiter -o fakes/sqlResult.go --fake-name SqlResult . sqlResult
type sqlResult interface {
	sql.Result
}
