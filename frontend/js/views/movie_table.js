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

define(['jquery', 'backbone', 'collections/movie_pageable', 'backgrid', 'views/movie_uri', 'backgrid_paginator', 'backgrid_filter'],
       function($, Backbone, PageableMovieCollection, Backgrid, MovieUri) {
         var MovieTableView = Backbone.View.extend({

           columns: [
             {
               name: "name",
               label: "Movie",
               editable: false,
               cell: MovieUri
             },
             {
               name: "downloads",
               label: "Downloads",
               editable: false,
               cell: "integer"
             }
           ],

           events: {
             'click #refreshButton': "_on_refreshbutton"
           },

           initialize: function() {
             // Create the grid, paginator, and filter for the movies
             // table. Attach them to the correct places in the DOM

             this.grid = new Backgrid.Grid({
               columns: this.columns,
               collection: new PageableMovieCollection()
             });
             this.$('#tableBox').append(this.grid.$el);

             this.paginator = new Backgrid.Extension.Paginator({
               collection: this.grid.collection
             });
             this.$('#paginatorBox').append(this.paginator.$el);

             this.filter = new Backgrid.Extension.ServerSideFilter({
               collection: this.grid.collection,
               placeholder: "Filter by name",
               fields: ['Name']
             });
             this.$('#filterBox').append(this.filter.$el);

             this.refresh();
           },

           refresh: function() {
             this.grid.collection.fetch({reset: true});
           },

           _on_refreshbutton: function() {
             // Goes back to page one before refreshing
             this.grid.collection.state.currentPage = 1;
             this.refresh();
           },

           render: function() {
             this.grid.render();
             this.paginator.render();
             this.filter.render();
           }

         });

         return MovieTableView;
       });
