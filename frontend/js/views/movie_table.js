/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
 */

/*
 * A view for the movie table. It basically manages a Backgrid grid,
 * paginator, filter, and reset button
 * exports: MovieTableView
 */

define(['jquery', 'underscore', 'backbone', 'collections/movie_pageable', 'backgrid', 'views/movie_uri', 'backgrid_paginator', 'backgrid_filter'],
       function($, _, Backbone, PageableMovieCollection, Backgrid, MovieUri) {
         var MovieTableView = Backbone.View.extend({

           templates: {
             tableSwitcher: _.template('<li><a href="#"><%= tableName %></a></li>')
           },

           columns: function(tableName) {
             return [
               {
                 name: "name",
                 label: "Movie",
                 editable: false,
                 cell: MovieUri(tableName)
               },
               {
                 name: "downloads",
                 label: "Downloads",
                 editable: false,
                 cell: "integer"
               }
             ];
           },

           events: {
             'click #refreshButton': "_on_refreshbutton"
           },

           tables: {},
           currentTable: null,
           eventItems: [],

           initialize: function(options) {
             // Create the grid, paginator, and filter for each of the
             // movie tables. Attach the current one to the correct
             // places in the DOM

             // Saves some of the common boxes
             this.tableKeysBox = this.$('#tableKeysBox');
             this.tableBox = this.$('#tableBox');
             this.paginatorBox = this.$('#paginatorBox');
             this.filterBox = this.$('#filterBox');
             this.noDataAlert = this.$('#no-data-alert');

             // Adds a clickable button for each collection, that
             // changes the currentTable to the named one
             _.each(options.tableKeys, _.bind(
               function(tableName) {
                 var switchButton = $(this.templates.tableSwitcher({ tableName: _.capitalize(tableName) }));
                 switchButton.on('click.handlers', _.partial(
                   function(outerThis) {
                     outerThis.tableKeysBox.children('li').removeClass('active');
                     $(this).addClass('active');
                     outerThis.currentTable = outerThis.tables[tableName];
                     outerThis.redraw();
                     outerThis.refresh();
                   }, this));
                 this.eventItems.push(switchButton);
                 this.tableKeysBox.append(switchButton);
               }, this));
             this.tableKeysBox.children('li:first-child').addClass('active');

             // Creates the grid, paginator, and filter for each
             // collection
             _.each(options.tableKeys, _.bind(
               function(tableName) {
                 var grid = new Backgrid.Grid({
                   columns: this.columns(tableName),
                   collection: new PageableMovieCollection([], { url: 'table/'+tableName })
                 });
                 var paginator = new Backgrid.Extension.Paginator({
                   collection: grid.collection
                 });
                 var filter = new Backgrid.Extension.ServerSideFilter({
                   collection: grid.collection,
                   placeholder: "Filter by name"
                 });

                 this.tables[tableName] = {
                   grid: grid,
                   paginator: paginator,
                   filter: filter
                 };

                 this.tableBox.append(grid.$el);
                 this.paginatorBox.append(paginator.$el);
                 this.filterBox.append(filter.$el);
               }, this));

             this.currentTable = this.tables[options.tableKeys[0]];
             this.redraw();
             this.refresh();
           },

           destroy: function() {
             // Removes the .handlers handlers on all items in
             // eventItems and empties the table from the DOM
             _.each(this.eventItems, function(item) {
               item.off('.handlers');
             });
             this.tableKeysBox.empty();
             this.tableBox.empty();
             this.paginatorBox.empty();
             this.filterBox.empty();
           },

           // Hides any elements in the filter, table, and paginator DOM positions
           hideEverything: function() {
             this.filterBox.children().hide();
             this.tableBox.children().hide();
             this.paginatorBox.children().hide();
           },

           // Shows the elements of the current table
           showCurrentTable: function() {
             this.currentTable.filter.$el.show();
             this.currentTable.grid.$el.show();
             this.currentTable.paginator.$el.show();
           },

           redraw: function() {
             this.hideEverything();
             this.showCurrentTable();
           },

           refresh: function() {
             // Re-fetches the current table's info
             this.currentTable.grid.collection.fetch({
               reset: true,
               success: _.bind(
                 function() {
                   if (this.currentTable.grid.collection.length === 0) {
                     this.noDataAlert.show();
                     this.hideEverything();
                     _.bind(this.refresh, this)();
                   } else {
                     this.noDataAlert.hide();
                     this.showCurrentTable();
                     this.render();
                   }
                 }, this)
             });
           },

           _on_refreshbutton: function() {
             // Goes back to page one before refreshing
             this.currentTable.grid.collection.state.currentPage = 1;
             this.refresh();
           },

           render: function() {
             this.currentTable.grid.render();
             this.currentTable.paginator.render();
             this.currentTable.filter.render();
           }

         });

         return MovieTableView;
       });