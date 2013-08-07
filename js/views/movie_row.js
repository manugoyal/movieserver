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
 * The view of a movies row in a movies table
 * exports: MovieRowView
 */

define(['underscore', 'backbone', 'text!views/templates/movie_row.html'],
       function(_, Backbone, MovieRowTemplate) {
         var MovieRowView = Backbone.View.extend({

           tagName: 'tr',

           initialize: function() {
             this.listenTo(this.model, 'change', this.render);
           },

           render: function() {
             var compiledTemplate = _.template(MovieRowTemplate, {movie: this.model});
             this.$el.html(compiledTemplate);
             return this;
           }

         });

         return MovieRowView;
       });
